package synapse

import (
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"sync"
	"time"
)

const PrometheusLabelWatch = "watch"

type WatcherZookeeper struct {
	WatcherCommon
	Hosts          []string
	Path           string
	TimeoutInMilli int

	connection       *nerve.SharedZkConnection
	connectionEvents <-chan zk.Event
}

func NewWatcherZookeeper() *WatcherZookeeper {
	w := &WatcherZookeeper{
		TimeoutInMilli: 2000,
	}
	return w
}

func (w *WatcherZookeeper) GetServiceName() string {
	return strings.Replace(w.Path, "/", "_", -1)[1:]
}

func (w *WatcherZookeeper) Init(service *Service) error {
	if err := w.CommonInit(service); err != nil {
		return errs.WithEF(err, w.fields, "Failed to init discovery")
	}
	w.fields = w.fields.WithField("path", w.Path)

	conn, err := nerve.NewSharedZkConnection(w.Hosts, time.Duration(w.TimeoutInMilli)*time.Millisecond)
	if err != nil {
		return errs.WithEF(err, w.fields, "Failed to prepare connection to zookeeper")
	}
	w.connection = conn
	w.connectionEvents = w.connection.Subscribe()
	return nil
}

func (w *WatcherZookeeper) Watch(context *ContextImpl, events chan<- ServiceReport, s *Service) {
	context.doneWaiter.Add(1)
	defer context.doneWaiter.Done()
	w.service.synapse.watcherFailures.WithLabelValues(w.service.Name, PrometheusLabelWatch).Set(0)

	reportsStop := make(chan struct{})
	go w.changedToReport(reportsStop, events, s)

	watcherStop := make(chan struct{})
	watcherStopWaiter := sync.WaitGroup{}
	go w.watchRoot(watcherStop, &watcherStopWaiter)

	<-context.stop
	logs.WithF(w.fields).Debug("Stopping watcher")
	close(watcherStop)
	watcherStopWaiter.Wait()
	w.connection.Close()
	close(reportsStop)
	logs.WithF(w.fields).Debug("Watcher stopped")
}

func (w *WatcherZookeeper) watchRoot(stop <-chan struct{}, doneWaiter *sync.WaitGroup) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()

	for {
		childs, _, rootEvents, err := w.connection.Conn.ChildrenW(w.Path)
		if err != nil {
			w.service.synapse.watcherFailures.WithLabelValues(w.service.Name, PrometheusLabelWatch).Inc()
			logs.WithEF(err, w.fields.WithField("path", w.Path)).Warn("Cannot watch root service path. Retry in 1s")
			<-time.After(time.Duration(1000) * time.Millisecond)

			if isStopped(stop) {
				return
			}
			continue
		}

		if len(childs) == 0 {
			w.reports.setNoNodes()
		} else {
			for _, child := range childs {
				if _, ok := w.reports.get(w.Path + "/" + child); !ok {
					go w.watchNode(w.Path+"/"+child, stop, doneWaiter)
				}
			}
		}

		select {
		case e := <-rootEvents:
			logs.WithF(w.fields.WithField("event", e)).Trace("Receiving event for root node")
			switch e.Type {
			case zk.EventNodeChildrenChanged | zk.EventNodeCreated | zk.EventNodeDataChanged | zk.EventNotWatching:
			// loop
			case zk.EventNodeDeleted:
				logs.WithF(w.fields.WithField("node", w.Path)).Debug("Rootnode deleted")
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
		content, stats, childEvent, err := w.connection.Conn.GetW(node)
		if err != nil {
			if err == zk.ErrNoNode {
				logs.WithEF(err, fields).Warn("Node disappear before watching")
				w.reports.removeNode(node)
				return
			}
			w.service.synapse.watcherFailures.WithLabelValues(w.service.Name, PrometheusLabelWatch).Inc()
			logs.WithEF(err, fields).Warn("Failed to watch node, retry in 1s")
			<-time.After(time.Duration(1000) * time.Millisecond)

			if isStopped(stop) {
				return
			}
			continue
		}

		w.reports.addRawReport(node, content, fields, stats.Ctime)

		select {
		case e := <-childEvent:
			logs.WithF(fields.WithField("event", e)).Trace("Receiving event from node")
			switch e.Type {
			case zk.EventNodeDataChanged | zk.EventNodeCreated | zk.EventNotWatching:
			// loop
			case zk.EventNodeDeleted:
				logs.WithF(fields).Debug("Node deleted")
				w.reports.removeNode(node)
				return
			}
		case <-stop:
			return
		}

	}
}

func isStopped(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
	}
	return false
}
