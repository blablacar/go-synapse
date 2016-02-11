package synapse

import (
	log "github.com/Sirupsen/logrus"
	"synapse/discovery"
	"time"
)

type SynapseService struct {
	Name string
	HAPPort int
	HAPServerOptions string
	HAPListen []string
	Discovery discovery.DiscoveryI
	KeepDefaultServers bool
	DefaultServers []SynapseServiceServerConfiguration
}

func(ss *SynapseService) Run(stop <-chan bool) error {
	defer servicesWaitGroup.Done()
	stopDiscovery := make(chan bool,1)
	go ss.Discovery.Run(stopDiscovery)
	Loop:
	for {
		select {
		case <-stop:
			stopDiscovery <- true
			break Loop
		default:
			time.Sleep(time.Second)
		}
	}
	ss.Discovery.WaitTermination()
	return nil
}

func(ss *SynapseService) Initialize(config SynapseServiceConfiguration,InstanceID string) error {
	ss.Name = config.Name
	ss.HAPPort = config.HAProxy.Port
	ss.HAPServerOptions = config.HAProxy.ServerOptions
	ss.HAPListen = config.HAProxy.Listen
	ss.DefaultServers = config.DefaultServers
	if config.KeepDefaultServers {
		ss.KeepDefaultServers = true
	}else {
		ss.KeepDefaultServers = false
	}
	var err error
	ss.Discovery, err = discovery.CreateDiscovery(config.Discovery.Type, 1000, config.Discovery.Path, config.Discovery.Hosts)
	if err != nil {
		log.WithError(err).Warn("Synapse Service [",ss.Name,"] Initilization fail")
		return err
	}
	return nil
}

func CreateService(config SynapseServiceConfiguration,InstanceID string) (*SynapseService, error) {
        var service SynapseService
        err := service.Initialize(config,InstanceID)
        if err != nil {
                log.WithError(err).Warn("Error Creating Service [",service.Name,"]")
		return nil, err
        }
        return &service, nil
}
