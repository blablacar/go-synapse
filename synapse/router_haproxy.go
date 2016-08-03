package synapse

import (
	"bytes"
	"encoding/json"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"strconv"
	"sync"
)

type RouterHaProxy struct {
	RouterCommon
	HaProxyClient
}
type HapRouterOptions struct {
	Frontend   []string
	Backend    []string
}
type HapServerOptions string

func NewRouterHaProxy() *RouterHaProxy {
	return &RouterHaProxy{}
}

func (r *RouterHaProxy) Run(stop chan struct{}, stopWaiter *sync.WaitGroup) {
	r.RunCommon(stop, stopWaiter, r)
}

func (r *RouterHaProxy) Init(s *Synapse) error {
	if err := r.commonInit(r, s); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to init common router")
	}
	if err := r.HaProxyClient.Init(); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to init haproxy client")
	}

	if r.ConfigPath == "" {
		return errs.WithF(r.RouterCommon.fields, "ConfigPath is required for haproxy router")
	}
	if len(r.ReloadCommand) == 0 {
		return errs.WithF(r.RouterCommon.fields, "ReloadCommand is required for haproxy router")
	}

	return nil
}

func (r *RouterHaProxy) isSameServers(report ServiceReport) bool {
	previous := r.lastEvents[report.service]

	if previous == nil || len(previous.reports) != len(report.reports) {
		logs.WithF(r.RouterCommon.fields).Debug("Number of Server has changed. Reload needed")
		return false
	}

	for _, new := range report.reports {
		found := false
		for _, old := range previous.reports {
			if new.Host == old.Host &&
				new.Port == old.Port &&
				new.Name == old.Name &&
				new.HaProxyServerOptions == old.HaProxyServerOptions {
				found = true
				break
			}
		}

		if !found {
			logs.WithF(r.RouterCommon.fields.WithField("server", new)).Debug("Server was not existing of options has changed")
			return false
		}
	}

	return true
}

func (r *RouterHaProxy) Update(serviceReport ServiceReport) error {
	r.ServerSort.Sort(&serviceReport.reports)

	front, back := r.toFrontendAndBackend(serviceReport)
	r.Frontend[serviceReport.service.Name] = front
	r.Backend[serviceReport.service.Name] = back

	if !r.isSameServers(serviceReport) || r.socketPath == "" {
		if err := r.Reload(); err != nil {
			return errs.WithEF(err, r.RouterCommon.fields, "Failed to reload haproxy")
		}
	} else if err := r.SocketUpdate(); err != nil {
		r.synapse.routerUpdateFailures.WithLabelValues(r.Type + "_socket").Inc()
		logs.WithEF(err, r.RouterCommon.fields).Error("Update by Socket failed. Reloading instead")
		if err := r.Reload(); err != nil {
			return errs.WithEF(err, r.RouterCommon.fields, "Failed to reload haproxy")
		}
	}
	return nil
}

func (r *RouterHaProxy) toFrontendAndBackend(serviceReport ServiceReport) ([]string, []string) {
	frontend := []string{}
	if serviceReport.service.typedRouterOptions != nil {
		for _, option := range serviceReport.service.typedRouterOptions.(HapRouterOptions).Frontend {
			frontend = append(frontend, option)
		}
	}
	frontend = append(frontend, "default_backend " + serviceReport.service.Name)

	backend := []string{}
	if serviceReport.service.typedRouterOptions != nil {
		for _, option := range serviceReport.service.typedRouterOptions.(HapRouterOptions).Backend {
			backend = append(backend, option)
		}
	}

	var serverOptions HapServerOptions
	if serviceReport.service.typedServerOptions != nil {
		serverOptions = serviceReport.service.typedServerOptions.(HapServerOptions)
	}
	for _, report := range serviceReport.reports {
		server := r.reportToHaProxyServer(report, serverOptions)
		backend = append(backend, server)
	}

	return frontend, backend
}

func (r *RouterHaProxy) reportToHaProxyServer(report Report, serverOptions HapServerOptions) string {
	var buffer bytes.Buffer
	buffer.WriteString("server ")
	buffer.WriteString(report.Name)
	buffer.WriteString(" ")
	buffer.WriteString(report.Host)
	buffer.WriteString(":")
	buffer.WriteString(strconv.Itoa(report.Port))
	buffer.WriteString(" ")
	if report.Weight != nil {
		buffer.WriteString("weight ")
		buffer.WriteString(strconv.Itoa(int(*report.Weight)))
	}
	buffer.WriteString(" ")
	buffer.WriteString(report.HaProxyServerOptions)
	buffer.WriteString(" ")
	buffer.WriteString(string(serverOptions))
	return buffer.String()
}

func (r *RouterHaProxy) ParseServerOptions(data []byte) (interface{}, error) {

	var serversOptions HapServerOptions
	err := json.Unmarshal(data, &serversOptions)
	if err != nil {
		return nil, errs.WithEF(err, r.RouterCommon.fields.WithField("content", data), "Failed to Unmarshal serverOptions")
	}
	return serversOptions, nil
}

func (r *RouterHaProxy) ParseRouterOptions(data []byte) (interface{}, error) {
	routerOptions := HapRouterOptions{}
	err := json.Unmarshal(data, &routerOptions)
	if err != nil {
		return nil, errs.WithEF(err, r.RouterCommon.fields.WithField("content", data), "Failed to Unmarshal routerOptions")
	}
	return routerOptions, nil
}
