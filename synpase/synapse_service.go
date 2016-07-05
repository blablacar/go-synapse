package synapse

import (
	log "github.com/Sirupsen/logrus"
	"github.com/blablacar/go-synapse/synpase/discovery"
	"time"
)

type SynapseService struct {
	Name                  string
	HAPPort               int
	HAPServerOptions      string
	HAPListen             []string
	HAPBackend            []string
	Discovery             discovery.DiscoveryI
	KeepDefaultServers    bool
	DefaultServers        []SynapseServiceServerConfiguration
	SharedFrontendName    string
	SharedFrontendContent []string
}

func (ss *SynapseService) Run(stop <-chan bool) error {
	defer servicesWaitGroup.Done()
	stopDiscovery := make(chan bool, 1)
	go ss.Discovery.Run(stopDiscovery)
Loop:
	for {
		select {
		case <-stop:
			log.Debug("SynapseService [", ss.Name, "] Stop Signal Received")
			stopDiscovery <- true
			break Loop
		default:
			time.Sleep(time.Second)
		}
	}
	log.Debug("SynapseService [", ss.Name, "] wait for Termination")
	ss.Discovery.WaitTermination()
	log.Debug("SynapseService [", ss.Name, "] terminated")
	return nil
}

func (ss *SynapseService) Initialize(config SynapseServiceConfiguration, InstanceID string, serviceModified chan bool) error {
	ss.Name = config.Name
	ss.HAPPort = config.HAProxy.Port
	ss.HAPServerOptions = config.HAProxy.ServerOptions
	ss.HAPListen = config.HAProxy.Listen
	ss.HAPBackend = config.HAProxy.Backend
	ss.DefaultServers = config.DefaultServers
	if config.KeepDefaultServers {
		ss.KeepDefaultServers = true
	} else {
		ss.KeepDefaultServers = false
	}
	var err error
	ss.Discovery, err = discovery.CreateDiscovery(config.Discovery.Type, 1000, config.Discovery.Path, config.Discovery.Hosts, serviceModified)
	if err != nil {
		log.WithError(err).Warn("Synapse Service [", ss.Name, "] Initilization fail")
		return err
	}
	if config.HAProxy.SharedFrontend.Name != "" {
		ss.SharedFrontendName = config.HAProxy.SharedFrontend.Name
		ss.SharedFrontendContent = config.HAProxy.SharedFrontend.Content
	}
	return nil
}

func CreateService(config SynapseServiceConfiguration, InstanceID string, serviceModified chan bool) (*SynapseService, error) {
	var service SynapseService
	err := service.Initialize(config, InstanceID, serviceModified)
	if err != nil {
		log.WithError(err).Warn("Error Creating Service [", service.Name, "]")
		return nil, err
	}
	return &service, nil
}
