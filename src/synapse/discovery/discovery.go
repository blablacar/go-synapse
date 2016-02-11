package discovery

import (
	log "github.com/Sirupsen/logrus"
	"errors"
	"strings"
)

type DiscoveredHost struct {
	ZKHostName string
	Name string `json:"name"`
	Host string `json:"host"`
	Port int `json:"port"`
	Maintenance bool `json:"maintenance"`
	Weight int `json:"weight"`
	HAProxyServerOptions string `json:"haproxy_server_options"`
	Tags []string `json:"tags"`
}

type Discovery struct {
	Type string
	ConnectTimeout int
	Hosts []DiscoveredHost
}

type DiscoveryI interface {
        GetType() string
        SetBaseConfiguration(ConnectTimeout int)
	GetDiscoveredHosts() []DiscoveredHost
	Run(stop <-chan bool) error
	Destroy() error
	WaitTermination()
}

// Create a Discovery object
// where:
// if Type == zookeeper
//      param1 is the path to watch for
//      param2 are the zk nodes to connect to
func CreateDiscovery(Type string, ConnectTimeout int, param1 string, param2 []string) (DiscoveryI, error) {
	var discovery DiscoveryI
        switch (strings.ToUpper(Type)) {
                case DISCOVERY_ZOOKEEPER_TYPE:
                        zookeeper_discovery := new(zookeeperDiscovery)
			zookeeper_discovery.Initialize()
			zookeeper_discovery.SetZKConfiguration(param2,param1)
			discovery = zookeeper_discovery
                default:
			err := errors.New("Unknown discovery type")
			log.Warn("Unknown discovery type [",Type,"]")
			return nil, err
        }
	discovery.SetBaseConfiguration(ConnectTimeout)
        return discovery, nil
}

func(d *Discovery) SetBaseConfiguration(ConnectTimeout int) {
	if ConnectTimeout > 0 {
		d.ConnectTimeout = ConnectTimeout
	}
}

func(d *Discovery) setType(Type string) {
	d.Type = Type
}

func(d *Discovery) GetDiscoveredHosts() ([]DiscoveredHost) {
	return d.Hosts
}
