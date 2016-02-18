package discovery

import (
	log "github.com/Sirupsen/logrus"
)

const DISCOVERY_BASE_TYPE string = "BASE"

type baseDiscovery struct {
	Discovery
}


func(bd *baseDiscovery) Initialize() {
	bd.Type = DISCOVERY_BASE_TYPE
}

func(bd *baseDiscovery) Run(stop <-chan bool) error {
	stopped := <-stop
	if stopped {
		log.Warn("Base Discovery, stopSignal Received")
	}else {
		log.Warn("Base Discovery, stopSignal Received, but ?? false ??")
	}
	return nil
}

func(zd *baseDiscovery) Destroy() error {
	return nil
}

func(zd *baseDiscovery) WaitTermination() {
	return
}

func(bd *baseDiscovery) GetType() string {
	return bd.Type
}
