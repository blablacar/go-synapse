package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"github.com/prometheus/client_golang/prometheus"
	"net"
)

type Synapse struct {
	LogLevel *logs.Level
	ApiHost  string
	ApiPort  int
	Routers  []json.RawMessage

	serviceAvailableCount   *prometheus.GaugeVec
	serviceUnavailableCount *prometheus.GaugeVec
	routerUpdateFailures    *prometheus.GaugeVec
	watcherFailures         *prometheus.GaugeVec

	fields           data.Fields
	synapseVersion   string
	synapseBuildTime string
	apiListener      net.Listener
	typedRouters     []Router
	context          *ContextImpl
}

func (s *Synapse) Init(version string, buildTime string, logLevelIsSet bool) error {
	s.synapseBuildTime = buildTime
	s.synapseVersion = version

	if s.ApiPort == 0 {
		s.ApiPort = 3455
	}

	if !logLevelIsSet && s.LogLevel != nil {
		logs.SetLevel(*s.LogLevel)
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

func (s *Synapse) Start(oneshot bool) error {
	logs.Info("Starting synapse")

	s.context = newContext(oneshot)
	for _, routers := range s.typedRouters {
		go routers.Run(s.context)
	}
	return s.startApi()
}

func (s *Synapse) Stop() {
	logs.Info("Stopping synapse")
	s.stopApi()
	close(s.context.stop)
	s.context.doneWaiter.Wait()
	logs.Debug("All router stopped")
}
