package synapse

import (
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/errs"
	"sync"
	"bytes"
	"strconv"
	"encoding/json"
)

type RouterHaProxy struct {
	RouterCommon
	HaProxyClient
}
type HapRouterOptions struct {
	Frontend []string
	Backend  []string
}
type HapServerOptions string


func NewRouterHaProxy() *RouterHaProxy {
	return &RouterHaProxy{}
}

func (r *RouterHaProxy) Start(stop chan struct{}, stopWaiter *sync.WaitGroup) {
	r.StartCommon(stop, stopWaiter, r)
}

func (r *RouterHaProxy) Init() error {
	if err := r.commonInit(r); err != nil {
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

func (r *RouterHaProxy) Update(serviceReport ServiceReport) error {
	front, back := r.toFrontendAndBackend(serviceReport)
	r.Frontend[serviceReport.service.Name] = front
	r.Backend[serviceReport.service.Name] = back

	if err := r.writeConfig(); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to write haproxy configuration")
	}
	if err := r.Reload(); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to reload haproxy")
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

func (r *RouterHaProxy) reportToHaProxyServer(report nerve.Report, serverOptions HapServerOptions) string {
	var buffer bytes.Buffer
	buffer.WriteString("server ")
	buffer.WriteString(report.Name)
	buffer.WriteString(" ")
	buffer.WriteString(report.Host)
	buffer.WriteString(":")
	buffer.WriteString(strconv.Itoa(report.Port))
	buffer.WriteString(" ")
	buffer.WriteString("weight ")
	buffer.WriteString(strconv.Itoa(int(report.Weight)))
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