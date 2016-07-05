package synapse

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/n0rad/go-erlog/errs"
	"sync"
	"github.com/blablacar/go-nerve/nerve"
	"time"
)

type WatcherZookeeper struct {
	WatcherCommon
	Hosts          []string
	Path           string
	TimeoutInMilli int

	connection     *zk.Conn
}

func NewWatcherZookeeper() *WatcherZookeeper {
	return &WatcherZookeeper{
		TimeoutInMilli: 2000,
	}
}

func (w *WatcherZookeeper) Init() error {
	if err := w.CommonInit(); err != nil {
		return errs.WithEF(err, w.fields, "Failed to init discovery")
	}
	w.fields = w.fields.WithField("path", w.Path)
	return nil
}

func (w *WatcherZookeeper) Watch(stop <-chan struct{}, doneWaiter *sync.WaitGroup, events chan<- []nerve.Report) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()

	for {
		_, _, err := zk.Connect(w.Hosts, time.Duration(w.TimeoutInMilli) * time.Millisecond)
		if err != nil {

		}


		select {
		case <- stop:
		// TODO stop
			return
		}
	}
}
