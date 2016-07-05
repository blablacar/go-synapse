package synapse

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"sync"
	"time"
)

var closeSynapseChan chan bool
var servicesWaitGroup sync.WaitGroup

func createServices(config SynapseConfiguration, servicesModified chan bool) ([]SynapseService, error) {
	var services []SynapseService
	if len(config.Services) > 0 {
		for i := 0; i < len(config.Services); i++ {
			service, err := CreateService(config.Services[i], config.InstanceID, servicesModified)
			if err != nil {
				log.Warn("Error when creating a service (", err, ")")
				return services, err
			}
			services = append(services, *service)
		}
	} else {
		err := errors.New("no service found in configuration")
		return services, err
	}
	return services, nil
}

func Run(stop <-chan bool, finished chan<- bool, synapseConfig SynapseConfiguration) {
	log.Debug("Synapse: Run function started")
	servicesModified := make(chan bool)
	services, err := createServices(synapseConfig, servicesModified)
	if err != nil {
		log.WithError(err).Warn("Services initiliazation failed, exiting")
		finished <- false
	} else {
		servicesWaitGroup.Add(len(services) + 1)
		stopper := make(chan bool, len(services)+1)
		//Start Discovery Services
		for i := 0; i < len(services); i++ {
			go services[i].Run(stopper)
		}
		//Start Output Management Routine
		var output SynapseOutput
		output.Initialize(synapseConfig.Output, services)
		go output.Run(stopper, servicesModified)

		// Wait for the stop signal
	Loop:
		for {
			select {
			case hasToStop := <-stop:
				if hasToStop {
					log.Debug("Synapse: Run function Close Signal Received")
				} else {
					log.Debug("Synapse: Run function Close Signal Received (but a strange false one)")
				}
				break Loop
			default:
				time.Sleep(time.Second * 1)
			}
		}

		//Inform all services and Haproxy routine to stop
		for i := 0; i < len(services)+1; i++ {
			stopper <- true
		}

		log.Debug("Synapse: Wait for all services to stop")
		//Wait for all services and Haproxy routine to shutdown
		servicesWaitGroup.Wait()
		finished <- true
	}
	log.Debug("Synapse: Run function termination")
}
