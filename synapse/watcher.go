package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"sync"
)

type WatcherCommon struct {
	Type string

	reports *reportMap
	service *Service
	fields  data.Fields
}

type Watcher interface {
	Init(service *Service) error
	GetFields() data.Fields
	Watch(stop <-chan struct{}, doneWaiter *sync.WaitGroup, events chan<- ServiceReport, s *Service)
	GetServiceName() string
}

func (w *WatcherCommon) CommonInit(service *Service) error {
	w.fields = data.WithField("type", w.Type)
	w.service = service
	w.reports = NewReportMap(service)
	return nil
}

func (w *WatcherCommon) GetFields() data.Fields {
	return w.fields
}

func WatcherFromJson(content []byte, service *Service) (Watcher, error) {
	t := &WatcherCommon{}
	if err := json.Unmarshal([]byte(content), t); err != nil {
		return nil, errs.WithE(err, "Failed to unmarshall watcher type")
	}

	fields := data.WithField("type", t.Type)
	var typedWatcher Watcher
	switch t.Type {
	case "zookeeper":
		typedWatcher = NewWatcherZookeeper()
	default:
		return nil, errs.WithF(fields, "Unsupported watcher type")
	}

	if err := json.Unmarshal([]byte(content), &typedWatcher); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to unmarshall watcher")
	}

	if err := typedWatcher.Init(service); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to init watcher")
	}
	return typedWatcher, nil
}

func (w *WatcherZookeeper) changedToReport(reportsStop <-chan struct{}, events chan<- ServiceReport, s *Service) {
	for {
		select {
		case <-w.reports.changed:
			reports := w.reports.getValues()
			events <- ServiceReport{Service: s, Reports: reports}
		case <-reportsStop:
			return
		}
	}
}
