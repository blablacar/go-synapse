package synapse
//
//import (
//	log "github.com/Sirupsen/logrus"
//	"time"
//	"github.com/n0rad/go-erlog/data"
//)
//
//type Service struct {
//	Name                  string
//	HAPPort               int
//	HAPServerOptions      string
//	HAPListen             []string
//	HAPBackend            []string
//	Discovery             Watcher
//	KeepDefaultServers    bool
//	DefaultServers        []SynapseServiceServerConfiguration
//	SharedFrontendName    string
//	SharedFrontendContent []string
//
//	fields                data.Fields
//}
//
//func (ss *Service) Run(stop <-chan bool) error {
//	defer servicesWaitGroup.Done()
//	stopDiscovery := make(chan bool, 1)
//	go ss.Discovery.Run(stopDiscovery)
//Loop:
//	for {
//		select {
//		case <-stop:
//			log.Debug("SynapseService [", ss.Name, "] Stop Signal Received")
//			stopDiscovery <- true
//			break Loop
//		default:
//			time.Sleep(time.Second)
//		}
//	}
//	log.Debug("SynapseService [", ss.Name, "] wait for Termination")
//	ss.Discovery.WaitTermination()
//	log.Debug("SynapseService [", ss.Name, "] terminated")
//	return nil
//}
//
//func (ss *Service) Initialize(config SynapseServiceConfiguration, serviceModified chan bool) error {
//	ss.Name = config.Name
//	ss.HAPPort = config.HAProxy.Port
//	ss.HAPServerOptions = config.HAProxy.ServerOptions
//	ss.HAPListen = config.HAProxy.Listen
//	ss.HAPBackend = config.HAProxy.Backend
//	ss.DefaultServers = config.DefaultServers
//	if config.KeepDefaultServers {
//		ss.KeepDefaultServers = true
//	} else {
//		ss.KeepDefaultServers = false
//	}
//	var err error
//	ss.Discovery, err = WatcherFromJson(config.Discovery.Type, 1000, config.Discovery.Path, config.Discovery.Hosts, serviceModified)
//	if err != nil {
//		log.WithError(err).Warn("Synapse Service [", ss.Name, "] Initilization fail")
//		return err
//	}
//	if config.HAProxy.SharedFrontend.Name != "" {
//		ss.SharedFrontendName = config.HAProxy.SharedFrontend.Name
//		ss.SharedFrontendContent = config.HAProxy.SharedFrontend.Content
//	}
//	return nil
//}
//
//func CreateService(config SynapseServiceConfiguration, serviceModified chan bool) (*Service, error) {
//	var service Service
//	err := service.Initialize(config, serviceModified)
//	if err != nil {
//		log.WithError(err).Warn("Error Creating Service [", service.Name, "]")
//		return nil, err
//	}
//	return &service, nil
//}
