package synapse

import (
	"text/template"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/data"
	"bufio"
	"bytes"
	"io/ioutil"
	"github.com/n0rad/go-erlog/logs"
	"github.com/blablacar/go-nerve/nerve"
)

const haProxyConfigurationTemplate = `# Handled by synapse. Do not modify it.
global
{{range .Global}}
  {{.}}{{end}}

defaults
{{range .Defaults}}
  {{.}}{{end}}

{{range $key, $element := .Listen}}
listen {{$key}}
{{range $element}}
  {{.}}{{end}}
{{end}}

{{range $key, $element := .Frontend}}
listen {{$key}}
{{range $element}}
  {{.}}{{end}}
{{end}}

{{range $key, $element := .Backend}}
listen {{$key}}
{{range $element}}
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
	ConfigPath           string
	SocketPath           string
	ReloadCommand        []string
	ReloadTimeoutInMilli int
	StatePath            string

	template             *template.Template
	fields               data.Fields
}

func (hap *HaProxyClient) Init() error {
	hap.fields = data.WithField("config", hap.ConfigPath)

	if hap.ReloadTimeoutInMilli == 0 {
		hap.ReloadTimeoutInMilli = 1000
	}

	tmpl, err := template.New("ha-proxy config").Parse(haProxyConfigurationTemplate)
	if err != nil {
		return errs.WithEF(err, hap.fields, "Failed to parse haproxy config template")
	}
	hap.template = tmpl

	return nil
}

func (hap *HaProxyClient) Reload() error {
	if err := nerve.ExecCommand(hap.ReloadCommand, hap.ReloadTimeoutInMilli); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to reload haproxy")
	}
	return nil
}

func (hap *HaProxyClient) writeConfig() error {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	if err := hap.template.Execute(writer, hap); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to ")
	}
	if err := writer.Flush(); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to flush buffer")
	}

	templated := b.Bytes()
	if logs.IsTraceEnabled() {
		logs.WithF(hap.fields.WithField("templated", templated)).Trace("Templated configuration file")
	}
	if err := ioutil.WriteFile(hap.ConfigPath, templated, 0644); err != nil {
		return errs.WithEF(err, hap.fields, "Failed to write configuration file")
	}
	return nil
}

