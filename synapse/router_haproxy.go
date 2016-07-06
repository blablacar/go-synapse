package synapse

import (
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/errs"
	"sync"
)

type RouterHaProxy struct {
	RouterCommon
	HaProxyClient
}

func NewRouterHaproxy() *RouterHaProxy {
	return &RouterHaProxy{}
}

func (r *RouterHaProxy) Start(stop chan struct{}, stopWaiter *sync.WaitGroup) {
	r.StartCommon(stop, stopWaiter, r)
}

func (r *RouterHaProxy) Init() error {
	if err := r.commonInit(); err != nil {
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

func (r *RouterHaProxy) Update(backend []nerve.Report) error {
	if err := r.writeConfig(); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to write haproxy configuration")
	}
	if err := r.Reload(); err != nil {
		return errs.WithEF(err, r.RouterCommon.fields, "Failed to reload haproxy")
	}
	return nil
}