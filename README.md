[![Build Status](https://travis-ci.org/blablacar/go-synapse.png?branch=master)](https://travis-ci.org/blablacar/go-synapse)

# Synapse

Synapse is a Service discovery mecanism. It watch servers for services in a backend and report status in a router.
This simplify service communication with backends and allow auto discovery & hot reconfiguration of the communication.
This provide better services discovery and faul-tolerant communication between services

At BlaBlaCar, we use a synapse for each service node that want to communicate with another service and discover those backend nodes (> 2000). [Nerve](https://github.com/blablacar/go-nerve) report node statuses to a Zookeeper and synapse watch it to update a local Hapoxy. All outgoing communication is going through this haproxy.

## Airbnb

Go-Synapse is a go rewrite of Airbnb's [Synapse](https://github.com/airbnb/synapse) with additiional features.

## Installation

Download the latest version on the [release page](https://github.com/blablacar/go-synapse/releases).

Create a configuration file base on the doc or [examples](https://github.com/blablacar/go-synapse/tree/master/examples).

Run with `./synapse synapse-config.yml`

### Building
_`****`_
Just clone the repository and run `./gomake`


## Configuration

It's a YAML file. You can find examples [here](https://github.com/blablacar/go-synapse/tree/master/examples)

Very minimal configuration file with only one service :
```yaml
routers:
  - type: console
    eventsBufferDurationInMilli: 500
    services:
      - serverSort: random          # random, name, date
        watcher:
          type: zookeeper
          hosts: ['localhost:2181']
          path: /services/api/myapi
        serverOptions:              
          ...                       # depend on router type
        routerOptions:              
          ...                       # depend on router type
```

Root attributes:

```yaml
logLevel: info
apiHost: 127.0.0.1
apiPort: 3454
routers:
    ...
```

### Router config

#### Router console

Nothing special to configure for this router.

```yaml
...
routers:
  - type: console
    services:
      - ...
```

#### Router haproxy

Router have haproxy specific attributes, but there is also services attributes 

```yaml
...
routers:
  - type: haproxy
    configPath: /tmp/hap.config
    reloadCommand: [./examples/haproxy_reload.sh]
    global:                                               # []string
      - stats   socket  /tmp/hap.socket level admin
    defaults:                                             # []string
    listen:                                               # map[string][]string
      stats:
         - mode http
         - bind 127.0.0.1:1936
         - stats enable

    services:
      - watcher:
          ...
        serverOptions: check inter 2s rise 3 fall 2
        routerOptions:
          frontend:
            - mode tcp
            - timeout client 31s
            - bind 127.0.0.1:5679
          backend:
            - mode tcp
            - timeout server 2m
            - timeout connect 45s
```

serverOptions support minimal templating:

```
serverOptions: cookie {{sha1String .Name}} check inter 2s rise 3 fall 2
serverOptions: cookie {{randString 10}} check inter 2s rise 3 fall 2
serverOptions: cookie {{.Name}} check inter 2s rise 3 fall 2
```

### Router template

```yaml
...
routers:
  - type: template
    destinationFile: /tmp/notexists/templated
    templateFile: ./examples/template.tmpl
    postTemplateCommand: [/bin/bash, -c, "echo 'ZZ' > /tmp/DDDD"]

    services:
      - watcher:
          ...
```


## Watcher config

### zookeeper watcher

```yaml

routers:
  - type: ...

    services:
        - watcher:
            type: zookeeper
            hosts: [ 'localhost:2181', 'localhost:2182' ]
            path: /services/es/es_site_search
            timeoutInMilli: 2000
                        
```

