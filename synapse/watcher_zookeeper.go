package synapse

import (
	"encoding/json"
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"github.com/samuel/go-zookeeper/zk"
	"sync"
	"time"
)

type nodes struct {
	sync.RWMutex
	m       map[string]nerve.Report
	changed chan struct{}
}

func NewNodes() nodes {
	n := nodes{}
	n.m = make(map[string]nerve.Report)
	n.changed = make(chan struct{})
	return n
}

func (n *nodes) addRawReport(name string, content []byte, failFields data.Fields) {
	report := nerve.Report{}
	if err := json.Unmarshal(content, &report); err != nil {
		logs.WithEF(err, failFields).Warn("Failed to unmarshal")
	}

	n.Lock()
	defer n.Unlock()
	n.m[name] = report
	n.changed <- struct{}{}
}

func (n *nodes) removeAll() {
	n.Lock()
	defer n.Unlock()
	for k := range n.m {
		delete(n.m, k)
	}
	n.changed <- struct{}{}

}

func (n *nodes) removeNode(name string) {
	n.Lock()
	defer n.Unlock()
	delete(n.m, name)
	n.changed <- struct{}{}
}

func (n *nodes) get(name string) (nerve.Report, bool) {
	n.RLock()
	defer n.RUnlock()
	value, ok := n.m[name]
	return value, ok
}

func (n *nodes) getValues() []nerve.Report {
	n.RLock()
	defer n.RUnlock()
	r := []nerve.Report{}
	for _, v := range n.m {
		r = append(r, v)
	}
	return r
}

/////////////////////////

type WatcherZookeeper struct {
	WatcherCommon
	Hosts            []string
	Path             string
	TimeoutInMilli   int

	reports          nodes
	connection       *zk.Conn
	connectionEvents <-chan zk.Event
}

func NewWatcherZookeeper() *WatcherZookeeper {
	w := &WatcherZookeeper{
		TimeoutInMilli: 2000,
		reports:        NewNodes(),
	}
	return w
}

func (w *WatcherZookeeper) Init() error {
	if err := w.CommonInit(); err != nil {
		return errs.WithEF(err, w.fields, "Failed to init discovery")
	}
	w.fields = w.fields.WithField("path", w.Path)

	conn, ev, err := zk.Connect(w.Hosts, time.Duration(w.TimeoutInMilli) * time.Millisecond)
	if err != nil {
		return errs.WithEF(err, w.fields, "Failed to prepare connection to zookeeper")
	}
	w.connection = conn
	w.connectionEvents = ev
	return nil
}

func (w *WatcherZookeeper) Watch(stop <-chan struct{}, doneWaiter *sync.WaitGroup, events chan <- []nerve.Report) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()
	watcherStop := make(chan struct{})
	watcherStopWaiter := sync.WaitGroup{}

	for {
		select {
		case <- w.reports.changed:
			events <- w.reports.getValues()
		case e := <-w.connectionEvents:
			logs.WithF(w.fields.WithField("event", e)).Trace("Receiving event for connection")
			switch e.Type {
			case zk.EventSession | zk.EventType(0):
				if e.State == zk.StateHasSession {
					go w.watchRoot(watcherStop, &watcherStopWaiter)
				}
			}
		case <-stop:
			close(watcherStop)
			watcherStopWaiter.Wait()
			w.connection.Close()
			return
		}
	}
}

func (w *WatcherZookeeper) watchRoot(stop <-chan struct{}, doneWaiter *sync.WaitGroup) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()

	for {
		childs, _, rootEvents, err := w.connection.ChildrenW(w.Path)
		if err != nil {
			logs.WithEF(err, w.fields.WithField("path", w.Path)).Warn("Cannot watch root service path")
			return
		}

		for _, child := range childs {
			if _, ok := w.reports.get(child); !ok {
				go w.watchNode(w.Path + "/" + child, stop, doneWaiter)
			}
		}

		select {
		case e := <-rootEvents:
			logs.WithF(w.fields.WithField("event", e)).Trace("Receiving event for root node")
			switch e.Type {
			case zk.EventNodeChildrenChanged | zk.EventNodeCreated | zk.EventNodeDataChanged | zk.EventNotWatching:
			// loop
			case zk.EventNodeDeleted:
				w.reports.removeAll()
			}
		case <-stop:
			return
		}
	}
}

func (w *WatcherZookeeper) watchNode(node string, stop <-chan struct{}, doneWaiter *sync.WaitGroup) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()

	fields := w.fields.WithField("node", node)
	logs.WithF(fields).Debug("New node watcher")

	for {
		content, _, childEvent, err := w.connection.GetW(node)
		if err != nil {
			logs.WithEF(err, w.fields).Warn("Failed to watch node")
			return
		}
		w.reports.addRawReport(node, content, fields)

		select {
		case e := <-childEvent:
			logs.WithF(fields.WithField("event", e)).Trace("Receiving event from node")
			switch e.Type {
			case zk.EventNodeDataChanged | zk.EventNodeCreated | zk.EventNotWatching:
			// loop
			case zk.EventNodeDeleted:
				w.reports.removeNode(node)
				return
			}
		case <-stop:
			return
		}

	}
}
