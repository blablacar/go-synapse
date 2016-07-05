## Discovery Classes

Discoverys are the piece of GO-Synapse that watch an external service registry
and reflect those changes in the local HAProxy state. Discoverys should conform
to the interface specified by `DiscoveryI`.
Here you can find the Base Discovery Implementation(file synapse/discovery/base.go):

```go
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

func(bd *baseDiscovery) Destroy() error {
	return nil
}

func(bd *baseDiscovery) WaitTermination() {
	return
}

func(bd *baseDiscovery) GetType() string {
	return bd.Type
}
```

Then you need to add a piece of code in the synapse/discovery/discovery.go file, in the function CreateDiscovery. For example here is the corresponding code for the base discovery:
```go
                case DISCOVERY_BASE_TYPE:
                        base_discovery := new(baseDiscovery)
                        base_discovery.Initialize()
                        discovery = base_discovery
```

### Discovery Specific Configuration

Unfortunatly, there's no automatic way of adding auto-discovered configuration for now. So you have to modify the synapse/synapse_configuration.go file. Then modify synapse/synapse_service.go (where the discovery object is created based on the configuration). And perhaps also modify the CreateDiscovery function in the synapse/discovery/discovery.go file.

### Discovery Plugin Inteface
Unlike the original Synapse from AirBNB, it's not possible to load your discovery as a plugin. You need all your code, when compiling synapse itself. 

<a name="backend_interface"/>
### Backend interface
Synapse understands the following fields in service backends (which are pulled
from the service registries):

`host` (string): The hostname of the service instance

`port` (integer): The port running the service on `host`

`name` (string, optional): The human readable name to refer to this service instance by

`weight` (float, optional): The weight that this backend should get when load
balancing to this service instance. Full support for updating HAProxy based on
this is still a WIP.

`haproxy_server_options` (string, optional): Any haproxy server options
specific to this particular server. They will be applied to the generated
`server` line in the HAProxy configuration. If you want Synapse to react to
changes in these lines you will need to enable the `state_file_path` option
in the main synapse configuration. In general the HAProxy backend level
`haproxy.server_options` setting is preferred to setting this per server
in your backends.

`maintenance` (string, optional): Sometimes, even if the service is working on a node, you want to exclude it from receiving connection. This tag is used in this case. For HAProxy output, it will disable the server, instead of removing it from the backend's server list
