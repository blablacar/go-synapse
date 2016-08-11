package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"sync"
	"time"
)

type RouterCommon struct {
	Type     string
	Services []*Service

	synapse    *Synapse
	lastEvents map[*Service]*ServiceReport
	fields     data.Fields
}

type Router interface {
	Init(s *Synapse) error
	getFields() data.Fields
	Run(stop chan struct{}, stopWaiter *sync.WaitGroup)
	Update(serviceReports []ServiceReport) error
	ParseServerOptions(data []byte) (interface{}, error)
	ParseRouterOptions(data []byte) (interface{}, error)
}

func (r *RouterCommon) commonInit(router Router, synapse *Synapse) error {
	r.fields = data.WithField("type", r.Type)
	r.synapse = synapse
	r.lastEvents = make(map[*Service]*ServiceReport)
	for _, service := range r.Services {
		if err := service.Init(router, synapse); err != nil {
			return errs.WithEF(err, r.fields, "Failed to init service")
		}
	}

	return nil
}

func (r *RouterCommon) RunCommon(stop chan struct{}, stopWaiter *sync.WaitGroup, router Router) {
	stopWaiter.Add(1)
	defer stopWaiter.Done()

	events := make(chan ServiceReport)
	watcherStop := make(chan struct{})
	watcherStopWaiter := sync.WaitGroup{}
	for _, service := range r.Services {
		go service.typedWatcher.Watch(watcherStop, &watcherStopWaiter, events, service)
	}

	go r.eventsProcessor(events, router)

	<-stop
	close(watcherStop)
	watcherStopWaiter.Wait()
	logs.WithF(r.fields).Debug("All Watchers stopped")
	close(events)
}

func (r *RouterCommon) eventsProcessor(events chan ServiceReport, router Router) {
	var firstUpdateDone bool
	var firstEventTimer *time.Timer
	var firstUpdateMutex sync.Mutex
	firstEvents := make(map[*Service]*ServiceReport)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			logs.WithF(r.fields.WithField("event", event)).Debug("Router received an event")

			firstUpdateMutex.Lock()
			if !firstUpdateDone {
				firstEvents[event.service] = &event
				if firstEventTimer == nil {
					firstEventTimer = time.AfterFunc(time.Second, func() {
						firstUpdateMutex.Lock()
						defer firstUpdateMutex.Unlock()
						reports := []ServiceReport{}
						for _, s := range firstEvents {
							reports = append(reports, *s)
						}

						r.handleReport(reports, router)
						firstEvents = nil
						firstEventTimer = nil
						firstUpdateDone = true
					})
				}
			} else {
				r.handleReport([]ServiceReport{event}, router)
			}
			firstUpdateMutex.Unlock()
		}
	}
}

func (r *RouterCommon) handleReport(events []ServiceReport, router Router) {
	validEvents := []ServiceReport{}

	for _, event := range events {
		event.service.ServerSort.Sort(&event.reports)

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
		validEvents = append(validEvents, event)
	}

	if len(validEvents) == 0 {
		logs.WithF(r.fields).Debug("Nothing to update on router")
		return
	}

	if err := router.Update(validEvents); err != nil {
		r.synapse.routerUpdateFailures.WithLabelValues(r.Type).Inc()
		logs.WithEF(err, r.fields).Error("Failed to report watch modification")
	}

	for _, e := range validEvents {
		r.lastEvents[e.service] = &e
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
