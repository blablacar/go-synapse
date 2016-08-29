package synapse

import (
	"encoding/json"
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/logs"
	"sync"
)

const PrometheusLabelContent = "content"

type reportMap struct {
	sync.RWMutex
	service *Service
	m       map[string]Report
	changed chan struct{}
}

type Report struct {
	nerve.Report
	CreationTime int64
}

func NewReportMap(service *Service) *reportMap {
	n := reportMap{
		service: service,
	}
	n.m = make(map[string]Report)
	n.changed = make(chan struct{})
	return &n
}

func (n *reportMap) setNoNodes() {
	n.Lock()
	n.m = make(map[string]Report)
	n.Unlock()
	n.changed <- struct{}{}
}

func (n *reportMap) addRawReport(name string, content []byte, failFields data.Fields, creationTime int64) {
	r := nerve.Report{}
	if err := json.Unmarshal(content, &r); err != nil {
		n.service.synapse.watcherFailures.WithLabelValues(n.service.Name, PrometheusLabelContent).Inc()
		logs.WithEF(err, failFields.WithField("content", string(content))).Warn("Failed to unmarshal report")
		return
	}
	n.Lock()
	n.m[name] = Report{r, creationTime}
	n.Unlock()
	n.changed <- struct{}{}
}

func (n *reportMap) removeAll() {
	n.Lock()
	for k := range n.m {
		delete(n.m, k)
	}
	n.Unlock()
	n.changed <- struct{}{}
}

func (n *reportMap) removeNode(name string) {
	n.Lock()
	delete(n.m, name)
	n.Unlock()
	n.changed <- struct{}{}
}

func (n *reportMap) get(name string) (Report, bool) {
	n.RLock()
	defer n.RUnlock()
	value, ok := n.m[name]
	return value, ok
}

func (n *reportMap) getValues() []Report {
	n.RLock()
	defer n.RUnlock()
	r := []Report{}
	for _, v := range n.m {
		r = append(r, v)
	}
	return r
}
