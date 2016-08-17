package synapse

import (
	"encoding/json"
	"fmt"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
)

type ServiceReport struct {
	service *Service
	reports []Report
}

func (s *ServiceReport) String() string {
	return s.service.Name + " " + fmt.Sprint(s.reports)
}

func (s *ServiceReport) HasActiveServers() bool {
	for _, report := range s.reports {
		if report.Available == nil || *report.Available {
			return true
		}
	}
	return false
}

func (s *ServiceReport) AvailableUnavailable() (int, int) {
	var available, unavailable int
	for _, report := range s.reports {
		if report.Available == nil || *report.Available {
			available++
		} else {
			unavailable++
		}
	}
	return available, unavailable
}

type Service struct {
	Name          string
	Watcher       json.RawMessage
	RouterOptions json.RawMessage
	ServerOptions json.RawMessage
	ServerSort    ReportSortType

	synapse            *Synapse
	fields             data.Fields
	typedWatcher       Watcher
	typedRouterOptions interface{}
	typedServerOptions interface{}
}

func (s *Service) Init(router Router, synapse *Synapse) error {
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
