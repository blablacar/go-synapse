package synapse

import (
	"bytes"
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"sync"
)

type ServiceReport struct {
	Service *Service
	Reports []Report
}

func (s ServiceReport) String() string {
	var buff bytes.Buffer
	buff.WriteString(s.Service.Name)
	buff.WriteRune('[')
	for i, r := range s.Reports {
		if i > 0 {
			buff.WriteString(", ")
		}
		buff.WriteRune('\'')
		buff.WriteString(r.String())
		buff.WriteRune('\'')
	}
	buff.WriteRune(']')
	return buff.String()
}

func (s *ServiceReport) HasActiveServers() bool {
	for _, report := range s.Reports {
		if report.Available == nil || *report.Available {
			return true
		}
	}
	return false
}

func (s *ServiceReport) AvailableUnavailable() (int, int) {
	var available, unavailable int
	for _, report := range s.Reports {
		if report.Available == nil || *report.Available {
			available++
		} else {
			unavailable++
		}
	}
	return available, unavailable
}

var idCount = 1
var idCountMutex = sync.Mutex{}

type ServerCorrelation struct {
	Type             string
	OtherServiceName string
	Scope            string

	otherService *Service
}

type Service struct {
	Name              string
	Watcher           json.RawMessage
	RouterOptions     json.RawMessage
	ServerOptions     json.RawMessage
	ServerSort        ReportSortType
	ServerCorrelation ServerCorrelation

	reported           bool
	id                 int
	router             Router
	synapse            *Synapse
	fields             data.Fields
	typedWatcher       Watcher
	typedRouterOptions interface{}
	typedServerOptions interface{}
}

func (s *Service) String() string {
	return s.Name
}

func (s *Service) Init(router Router, synapse *Synapse) error {
	idCountMutex.Lock()
	s.id = idCount
	idCount++
	idCountMutex.Unlock()

	s.router = router
	s.synapse = synapse
	s.fields = router.getFields().WithField("service", s.Name)
	watcher, err := WatcherFromJson(s.Watcher, s)
	if err != nil {
		return errs.WithEF(err, s.fields, "Failed to read watcher")
	}
	logs.WithF(watcher.GetFields()).Debug("Watcher loaded")
	s.typedWatcher = watcher
	if err := s.typedWatcher.Init(s); err != nil {
		return errs.WithEF(err, s.fields, "Failed to init watcher")
	}

	if s.ServerCorrelation.Type != "" {
		if s.ServerCorrelation.Type != "excludeServer" {
			return errs.WithF(s.fields.WithField("type", s.ServerCorrelation.Type), "Unsupported serverCorrelation type")
		}
		if s.ServerCorrelation.Scope != "first" {
			return errs.WithF(s.fields.WithField("scope", s.ServerCorrelation.Scope), "Unsupported serverCorrelation scope")
		}
		if os, err := s.router.GetService(s.ServerCorrelation.OtherServiceName); err != nil {
			return errs.WithF(s.fields.WithField("otherServiceName", s.ServerCorrelation.OtherServiceName), "Other service not found for this name")
		} else {
			s.ServerCorrelation.otherService = os
		}
	}

	if s.Name == "" {
		s.Name = s.typedWatcher.GetServiceName()
		s.fields = s.fields.WithField("service", s.Name)
	}

	if len([]byte(s.RouterOptions)) > 0 {
		typedRouterOptions, err := router.ParseRouterOptions(s.RouterOptions)
		if err != nil {
			return errs.WithEF(err, s.fields, "Failed to parse routerOptions")
		}
		s.typedRouterOptions = typedRouterOptions
	}

	if len([]byte(s.RouterOptions)) > 0 {
		typedServerOptions, err := router.ParseServerOptions(s.ServerOptions)
		if err != nil {
			return errs.WithEF(err, s.fields, "Failed to parse serverOptions")
		}
		s.typedServerOptions = typedServerOptions
	}

	if s.ServerSort == "" {
		s.ServerSort = SORT_RANDOM
	}

	logs.WithF(s.fields).Info("Service loaded")
	logs.WithF(s.fields.WithField("data", s)).Debug("Service loaded")
	return nil
}
