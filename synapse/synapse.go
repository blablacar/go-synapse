package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"github.com/prometheus/client_golang/prometheus"
	"net"
	"sync"
)

type Synapse struct {
	ApiHost string
	ApiPort int
	Routers []json.RawMessage

	serviceAvailableCount   *prometheus.GaugeVec
	serviceUnavailableCount *prometheus.GaugeVec
	routerUpdateFailures    *prometheus.GaugeVec
	watcherFailures         *prometheus.GaugeVec

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

	if s.ApiPort == 0 {
		s.ApiPort = 3455
	}

	s.routerUpdateFailures = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "synapse",
			Name:      "router_update_failure",
			Help:      "router update failures",
		}, []string{"type"})

	s.serviceAvailableCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "synapse",
			Name:      "service_available_count",
			Help:      "service available status",
		}, []string{"service"})

	s.serviceUnavailableCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "synapse",
			Name:      "service_unavailable_count",
			Help:      "service unavailable status",
		}, []string{"service"})

	s.watcherFailures = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "synapse",
			Name:      "watcher_failure",
			Help:      "watcher failure",
		}, []string{"service", "type"})

	if err := prometheus.Register(s.watcherFailures); err != nil {
		return errs.WithEF(err, s.fields, "Failed to register prometheus watcher_failure")
	}

	if err := prometheus.Register(s.serviceAvailableCount); err != nil {
		return errs.WithEF(err, s.fields, "Failed to register prometheus service_available_count")
	}

	if err := prometheus.Register(s.serviceUnavailableCount); err != nil {
		return errs.WithEF(err, s.fields, "Failed to register prometheus service_unavailable_count")
	}

	if err := prometheus.Register(s.routerUpdateFailures); err != nil {
		return errs.WithEF(err, s.fields, "Failed to register prometheus router_update_failure")
	}

	for _, data := range s.Routers {
		router, err := RouterFromJson(data, s)
		if err != nil {
			return errs.WithE(err, "Failed to init router")
		}
		s.typedRouters = append(s.typedRouters, router)
	}

	return nil
}

func (s *Synapse) Start(startStatus chan<- error) {
	logs.Info("Starting synapse")
	for _, routers := range s.typedRouters {
		go routers.Run(s.routerStopper, &s.routerStopWait)
	}
	res := s.startApi()
	if startStatus != nil {
		startStatus <- res
	}
}

func (s *Synapse) Stop() {
	logs.Info("Stopping synapse")
	s.stopApi()
	close(s.routerStopper)
	s.routerStopWait.Wait()
	logs.Debug("All router stopped")
}
