package synapse

import (
	"encoding/json"
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"sync"
	"github.com/n0rad/go-erlog/logs"
)

type RouterCommon struct {
	Type     string
	Services []*Service

	fields data.Fields
}

type Router interface {
	Init() error
	getFields() data.Fields
	Start(stop chan struct{}, stopWaiter *sync.WaitGroup)
	Update(backends []nerve.Report) error
}

func (r *RouterCommon) commonInit() error {
	r.fields = data.WithField("type", r.Type)
	for _, service := range r.Services {
		if err := service.Init(); err != nil {
			return errs.WithEF(err, r.fields, "Failed to init service")
		}
	}

	return nil
}

func (r *RouterCommon) StartCommon(stop chan struct{}, stopWaiter *sync.WaitGroup, router Router) {
	stopWaiter.Add(1)
	defer stopWaiter.Done()

	events := make(chan []nerve.Report)

	for _, service := range r.Services {
		go service.typedWatcher.Watch(stop, stopWaiter, events)
	}

	for {
		select {
		case event := <-events:
			if err := router.Update(event); err != nil {
				logs.WithEF(err, r.fields).Error("Failed to report watch modification")
			}
		case <-stop:
			return
		}
	}
}

func (r *RouterCommon) getFields() data.Fields {
	return r.fields
}

func RouterFromJson(content []byte) (Router, error) {
	t := &RouterCommon{}
	if err := json.Unmarshal([]byte(content), t); err != nil {
		return nil, errs.WithE(err, "Failed to unmarshall check type")
	}

	fields := data.WithField("type", t.Type)
	var typedRouter Router
	switch t.Type {
	case "console":
		typedRouter = NewRouterConsole()
	case "haproxy":
		typedRouter = NewRouterHaproxy()
	default:
		return nil, errs.WithF(fields, "Unsupported router type")
	}

	if err := json.Unmarshal([]byte(content), &typedRouter); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to unmarshall router")
	}

	if err := typedRouter.Init(); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to init router")
	}
	return typedRouter, nil
}
