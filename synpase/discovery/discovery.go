package discovery

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"strings"
)

type DiscoveredHost struct {
	ZKHostName           string
	Name                 string   `json:"name"`
	Host                 string   `json:"host"`
	Port                 int      `json:"port"`
	Maintenance          bool     `json:"maintenance"`
	Weight               int      `json:"weight"`
	HAProxyServerOptions string   `json:"haproxy_server_options"`
	Tags                 []string `json:"tags"`
}

type Discovery struct {
	Type            string
	ConnectTimeout  int
	Hosts           []DiscoveredHost
	serviceModified chan bool
}

type DiscoveryI interface {
	GetType() string
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
func CreateDiscovery(Type string, ConnectTimeout int, param1 string, param2 []string, serviceModified chan bool) (DiscoveryI, error) {
	var discovery DiscoveryI
	switch strings.ToUpper(Type) {
	case DISCOVERY_ZOOKEEPER_TYPE:
		zookeeper_discovery := new(zookeeperDiscovery)
		zookeeper_discovery.Initialize()
		zookeeper_discovery.SetZKConfiguration(param2, param1)
		zookeeper_discovery.serviceModified = serviceModified
		zookeeper_discovery.SetBaseConfiguration(ConnectTimeout)
		discovery = zookeeper_discovery
	case DISCOVERY_BASE_TYPE:
		base_discovery := new(baseDiscovery)
		base_discovery.Initialize()
		discovery = base_discovery
	default:
		err := errors.New("Unknown discovery type")
		log.Warn("Unknown discovery type [", Type, "]")
		return nil, err
	}
	return discovery, nil
}

func (d *Discovery) SetBaseConfiguration(ConnectTimeout int) {
	if ConnectTimeout > 0 {
		d.ConnectTimeout = ConnectTimeout
	}
}

func (d *Discovery) setType(Type string) {
	d.Type = Type
}

func (d *Discovery) GetDiscoveredHosts() []DiscoveredHost {
	return d.Hosts
}
