package synapse

import (
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"encoding/json"
	"sync"
	"github.com/blablacar/go-nerve/nerve"
)

type WatcherCommon struct {
	Type            string

	fields data.Fields
}

type Watcher interface {
	Init() error
	GetFields() data.Fields
	Run(stop <-chan bool, doneWaiter *sync.WaitGroup, events chan<- []nerve.Report)
}

func (w *WatcherCommon) CommonInit() error {
	w.fields = data.WithField("type", w.Type)
	return nil
}

func (w *WatcherCommon) GetFields() data.Fields {
	return w.fields
}

func WatcherFromJson(content []byte) (Watcher, error) {
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

	if err := typedWatcher.Init(); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to init watcher")
	}
	return typedWatcher, nil
}