package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"sync"
	"github.com/n0rad/go-erlog/logs"
)

type RouterCommon struct {
	Type      string
	Services  []*Service

	synapse   *Synapse
	lastEvents map[*Service]*ServiceReport
	fields    data.Fields
}

type Router interface {
	Init(s *Synapse) error
	getFields() data.Fields
	Start(stop chan struct{}, stopWaiter *sync.WaitGroup)
	Update(serviceReport ServiceReport) error
	ParseServerOptions(data []byte) (interface{}, error)
	ParseRouterOptions(data []byte) (interface{}, error)
}

func (r *RouterCommon) commonInit(router Router, s *Synapse) error {
	r.fields = data.WithField("type", r.Type)
	r.synapse = s
	r.lastEvents = make(map[*Service]*ServiceReport)
	for _, service := range r.Services {
		if err := service.Init(router); err != nil {
			return errs.WithEF(err, r.fields, "Failed to init service")
		}
	}

	return nil
}

func (r *RouterCommon) StartCommon(stop chan struct{}, stopWaiter *sync.WaitGroup, router Router) {
	stopWaiter.Add(1)
	defer stopWaiter.Done()

	events := make(chan ServiceReport)

	for _, service := range r.Services {
		go service.typedWatcher.Watch(stop, stopWaiter, events, service)
	}

	for {
		select {
		case event := <-events:
			logs.WithF(r.fields.WithField("event", event)).Debug("Router received an event")
			available, unavailable := event.AvailableUnavailable()
			r.synapse.serviceAvailableCount.WithLabelValues(event.service.Name).Set(float64(available))
			r.synapse.serviceUnavailableCount.WithLabelValues(event.service.Name).Set(float64(unavailable))
			if !event.HasActiveServers() {
				if r.lastEvents[event.service] == nil {
					logs.WithF(event.service.fields).Warn("First Report has no active server. Not declaring in router")
				} else {
					logs.WithF(event.service.fields).Error("Receiving report with no active server. Keeping previous report")
				}
				continue
			} else if r.lastEvents[event.service] == nil || r.lastEvents[event.service].HasActiveServers() != event.HasActiveServers() {
				logs.WithF(event.service.fields.WithField("event", event)).Info("Server(s) available for router")
			}
			if err := router.Update(event); err != nil {
				logs.WithEF(err, r.fields).Error("Failed to report watch modification")
			}
			r.lastEvents[event.service] = &event
		case <-stop:
			return
		}
	}
}

func (r *RouterCommon) getFields() data.Fields {
	return r.fields
}

func RouterFromJson(content []byte, s *Synapse) (Router, error) {
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
		typedRouter = NewRouterHaProxy()
	default:
		return nil, errs.WithF(fields, "Unsupported router type")
	}

	if err := json.Unmarshal([]byte(content), &typedRouter); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to unmarshall router")
	}

	if err := typedRouter.Init(s); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to init router")
	}
	return typedRouter, nil
}
