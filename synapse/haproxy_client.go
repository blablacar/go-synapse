package synapse

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
)

const haProxyConfigurationTemplate = `# Handled by synapse. Do not modify it.
global
{{- range .Global}}
  {{.}}{{end}}

defaults
{{- range .Defaults}}
  {{.}}{{end}}

{{range $key, $element := .Listen}}
listen {{$key}}
{{- range $element}}
  {{.}}{{end}}
{{end}}
{{range $key, $element := .Frontend}}
frontend {{$key}}
{{- range $element}}
  {{.}}{{end}}
{{end}}
{{range $key, $element := .Backend}}
backend {{$key}}
{{- range $element}}
  {{.}}{{end}}
{{end}}

`

type HaProxyConfig struct {
	Global   []string
	Defaults []string
	Listen   map[string][]string
	Frontend map[string][]string
	Backend  map[string][]string
}

type HaProxyClient struct {
	HaProxyConfig
	ConfigPath               string
	ReloadCommand            []string
	ReloadMinIntervalInMilli int
	ReloadTimeoutInMilli     int
	StatePath                string
	CleanupCommand           []string
	CleanupTimeoutInMilli    int

	reloadMutex   sync.Mutex
	socketPath    string
	weightRegex   *regexp.Regexp
	enabledRegex  *regexp.Regexp
	disabledRegex *regexp.Regexp
	lastReload    time.Time
	template      *template.Template
	fields        data.Fields
}

func (hap *HaProxyClient) Init() error {
	hap.fields = data.WithField("config", hap.ConfigPath)

	if hap.Listen == nil {
		hap.Listen = make(map[string][]string)
	}
	if hap.Frontend == nil {
		hap.Frontend = make(map[string][]string)
	}
	if hap.Backend == nil {
		hap.Backend = make(map[string][]string)
	}

	if hap.ReloadMinIntervalInMilli == 0 {
		hap.ReloadMinIntervalInMilli = 500
	}

	if hap.ReloadTimeoutInMilli == 0 {
		hap.ReloadTimeoutInMilli = 1000
	}
	if hap.CleanupTimeoutInMilli == 0 {
		hap.CleanupTimeoutInMilli = 35 * 1000
	}

	hap.weightRegex = regexp.MustCompile(`server[\s]+([\S]+).*weight[\s]+([\d]+)`)
	//hap.enabledRegex = regexp.MustCompile(`server[\s]+([\S]+).*enabled[\s]?`)
	hap.enabledRegex = regexp.MustCompile(`server\s+(\S+)\s+(\d+\.\d+\.\d+\.\d+):(\d+).*enabled\s?`)
	hap.disabledRegex = regexp.MustCompile(`server[\s]+([\S]+).*disabled[\s]?`)

	hap.socketPath = hap.findSocketPath()
	if hap.socketPath == "" {
		logs.WithF(hap.fields).Warn("No socketPath file specified. Will update by reload only")
	}

	tmpl, err := template.New("ha-proxy-config").Parse(haProxyConfigurationTemplate)
	if err != nil {
		return errs.WithEF(err, hap.fields, "Failed to parse haproxy config template")
	}
	hap.template = tmpl

	return nil
}

func (hap *HaProxyClient) findSocketPath() string {
	socketRegex := regexp.MustCompile(`stats[\s]+socket[\s]+(\S+)`)
	for _, str := range hap.Global {
		res := socketRegex.FindStringSubmatch(str)
		if len(res) > 1 {
			return res[1]
		}
	}
	return ""
}

func (hap *HaProxyClient) Reload() error {
	hap.reloadMutex.Lock()
	defer hap.reloadMutex.Unlock()

	if err := hap.writeConfig(); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to write haproxy configuration")
	}

	logs.WithF(hap.fields).Info("Reloading haproxy")
	env := append(os.Environ(), "HAP_CONFIG="+hap.ConfigPath)

	waitDuration := hap.lastReload.Add(time.Duration(hap.ReloadMinIntervalInMilli) * time.Millisecond).Sub(time.Now())
	if waitDuration > 0 {
		logs.WithF(hap.fields.WithField("wait", waitDuration)).Debug("Reloading too fast")
		time.Sleep(waitDuration)
	}
	defer func() {
		hap.lastReload = time.Now()
	}()
	if err := nerve.ExecCommandFull(hap.ReloadCommand, env, hap.ReloadTimeoutInMilli); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to reload haproxy")
	}
	if len(hap.CleanupCommand) > 0 {
		go func() {
			if err := nerve.ExecCommandFull(hap.CleanupCommand, env, hap.CleanupTimeoutInMilli); err != nil {
				logs.WithEF(err, hap.fields).Warn("Cleanup command failed")
			}
		}()
	}
	return nil
}

func (hap *HaProxyClient) SocketUpdate() error {
	if hap.socketPath == "" {
		return errs.WithF(hap.fields, "No socket file specified. Cannot update")
	}
	logs.WithF(hap.fields).Debug("Updating haproxy by socket")

	if err := hap.writeConfig(); err != nil { // just to stay in sync
		logs.WithEF(err, hap.fields).Warn("Failed to write configuration file")
	}

	conn, err := net.Dial("unix", hap.socketPath)
	if err != nil {
		return errs.WithEF(err, hap.fields.WithField("socket", hap.socketPath), "Failed to connect to haproxy socket")
	}
	defer conn.Close()

	i := 0
	b := bytes.Buffer{}
	for name, servers := range hap.Backend {
		for _, server := range servers {
			res := hap.weightRegex.FindStringSubmatch(server)
			if len(res) == 3 {
				i++
				b.WriteString(fmt.Sprintf("set server %s/%s weight %s\n", name, res[1], res[2]))
			}

			res = hap.enabledRegex.FindStringSubmatch(server)
			if len(res) == 4 {
				i++
				b.WriteString(fmt.Sprintf("set server %s/%s state ready\n", name, res[1]))
				b.WriteString(fmt.Sprintf("set server %s/%s addr %s %s\n", name, res[1], res[2], res[3]))
			}
			res = hap.disabledRegex.FindStringSubmatch(server)
			if len(res) == 2 {
				i++
				b.WriteString(fmt.Sprintf("set server %s/%s state maint\n", name, res[1]))
			}

		}
	}

	if b.Len() == 0 {
		logs.WithF(hap.fields).Debug("Nothing to update by socket. No weight set")
		return nil
	}

	commands := b.Bytes()

	logs.WithF(hap.fields.WithField("command", string(commands))).Trace("Running command on hap socket")
	count, err := conn.Write(commands)
	if count != len(commands) || err != nil {
		return errs.WithEF(err, hap.fields.
			WithField("written", count).
			WithField("len", len(commands)).
			WithField("command", string(commands)), "Failed to write command to haproxy")
	}

	scanner := bufio.NewScanner(conn)
	updateFailed := false
	line := ""
	for scanner.Scan() {
		line = scanner.Text()
		if line != "" && !strings.HasPrefix(line, "no need to change") {
			updateFailed = true
			break
		}

	}
	if updateFailed {
		return errs.WithF(hap.fields.WithField("response", line), "Bad response for haproxy socket command")
	}
	if err := scanner.Err(); err != nil {
		return errs.WithF(hap.fields.WithField("response", line), "Bad response for haproxy socket command")
	}

	return nil
}

func (hap *HaProxyClient) writeConfig() error {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	if err := hap.template.Execute(writer, hap); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to template haproxy configuration file")
	}
	if err := writer.Flush(); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to flush buffer")
	}

	templated := b.Bytes()
	if logs.IsTraceEnabled() {
		logs.WithF(hap.fields.WithField("templated", string(templated))).Trace("Templated configuration file")
	}
	if err := ioutil.WriteFile(hap.ConfigPath, templated, 0644); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to write configuration file")
	}
	return nil
}
