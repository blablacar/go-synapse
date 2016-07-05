package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"net"
	"github.com/n0rad/go-erlog/data"
	"sync"
)

type Synapse struct {
	ApiHost          string
	ApiPort          int
	Routers          []json.RawMessage

	fields           data.Fields
	synapseVersion   string
	synapseBuildTime string
	apiListener      net.Listener
	typedRouters     []Router
	routerStopper    chan struct{}
	routerStopWait   sync.WaitGroup
}

func (s *Synapse) Init(version string, buildTime string) error {
	s.synapseBuildTime = buildTime
	s.synapseVersion = version
	s.routerStopper = make(chan struct{})

	for _, data := range s.Routers {
		router, err := RouterFromJson(data)
		if err != nil {
			return errs.WithE(err, "Failed to init router")
		}
		s.typedRouters = append(s.typedRouters, router)
	}

	return nil
}

func (s *Synapse) Start(startStatus chan<-error) {
	logs.Info("Starting synapse")
	for _, routers := range s.typedRouters {
		go routers.Start(s.routerStopper)
	}
	res := s.startApi()
	if startStatus != nil {
		startStatus <- res
	}
}

func (s *Synapse) Stop() {
	logs.Info("Stopping synapse")
	close(s.routerStopper)
	s.stopApi()
	s.routerStopWait.Wait()
	logs.Debug("All router stopped")
}
