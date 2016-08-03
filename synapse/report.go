package synapse

import (
	"encoding/json"
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/logs"
	"sync"
)

type reportMap struct {
	sync.RWMutex
	m       map[string]Report
	changed chan struct{}
}

type Report struct {
	nerve.Report
	CreationTime int64
}

func NewNodes() reportMap {
	n := reportMap{}
	n.m = make(map[string]Report)
	n.changed = make(chan struct{}, 100) // zklib sux and will drop events if chan is full
	return n
}

func (n *reportMap) setNoNodes() {
	n.Lock()
	defer n.Unlock()
	n.m = make(map[string]Report)
	n.changed <- struct{}{}
}

func (n *reportMap) addRawReport(name string, content []byte, failFields data.Fields, creationTime int64) {
	r := nerve.Report{}
	if err := json.Unmarshal(content, &r); err != nil {
		logs.WithEF(err, failFields).Warn("Failed to unmarshal report")
	}

	n.Lock()
	defer n.Unlock()
	n.m[name] = Report{r, creationTime}
	n.changed <- struct{}{}
}

func (n *reportMap) removeAll() {
	n.Lock()
	defer n.Unlock()
	for k := range n.m {
		delete(n.m, k)
	}
	n.changed <- struct{}{}

}

func (n *reportMap) removeNode(name string) {
	n.Lock()
	defer n.Unlock()
	delete(n.m, name)
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

