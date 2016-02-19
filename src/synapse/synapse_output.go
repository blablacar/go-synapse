package synapse

import (
	log "github.com/Sirupsen/logrus"
	"synapse/output"
	"sort"
)

//A wrapper between output and the rest of Synapse
//It transform SYnapse COnfiguration into ouput args
//And manages signal to transform services into an OutputBackendSlice
type SynapseOutput struct {
	Output output.OutputI
	Services []SynapseService
}

func(so *SynapseOutput) Run(stopper chan bool,servicesModified chan bool) {
	defer servicesWaitGroup.Done()
	backendsChan := make(chan output.OutputBackendSlice)
	so.Output.Initialize()
	go so.Output.Run(backendsChan)
	Loop:
	for {
		select {
			case <-stopper:
				//Need to stop output
				so.Output.Stop()
				break Loop
			case <-servicesModified:
				backends := so.GetAllBackends()
				backendsChan <- backends
		}
	}
	so.Output.WaitTermination()
	log.Debug("Stopping Synapse Output Wrapper")
}

func(so *SynapseOutput) Initialize(config SynapseOutputConfiguration, services []SynapseService) {
	so.Services = services
	switch config.Type {
	case "haproxy":
		so.Output = so.createHAProxyOutput(config)
	case "file":
		so.Output = so.createFileOutput(config)
	}
	//First Initialisation of backends
	so.Output.SetBackends(so.GetAllBackends())
}

func (so *SynapseOutput) createHAProxyOutput(config SynapseOutputConfiguration) output.OutputI {
	var sharedFrontends []output.HAProxyOutputSharedFrontend
	for _, sharedFrontend := range config.SharedFrontend {
		var hapSF output.HAProxyOutputSharedFrontend
		hapSF.Name = sharedFrontend.Name
		hapSF.Content = sharedFrontend.Content
		sharedFrontends = append(sharedFrontends,hapSF)
	}
	return output.CreateOutput(
		config.Type,
		config.ConfigFilePath,
		config.DoWrites,
		config.DoReloads,
		config.DoSocket,
		config.Global,
		config.Defaults,
		config.ReloadCommand.Binary,
		config.ReloadCommand.Arguments,
		config.SocketFilePath,
		config.RestartInterval,
		config.StateFilePath,
		config.StateFileTTL,
		config.BindAddress,
		sharedFrontends)
}

func (so *SynapseOutput) createFileOutput(config SynapseOutputConfiguration) output.OutputI {
	return output.CreateOutput(config.Type,config.OutputFilePath,true,false,false,nil,nil,"",nil,"",0,"",0,"",nil)
}

func(so *SynapseOutput) GetAllBackends() output.OutputBackendSlice {
	var backends output.OutputBackendSlice
	for _, service := range so.Services {
		var backend output.OutputBackend
		backend.Name = service.Name
		backend.Port = service.HAPPort
		backend.ServerOptions = service.HAPServerOptions
		backend.Listen = service.HAPListen
		backend.Backend = service.HAPBackend
		backend.SharedFrontendName = service.SharedFrontendName
		backend.SharedFrontendContent = service.SharedFrontendContent
		//Get All dynamic servers to include
		discoveredHosts := service.Discovery.GetDiscoveredHosts()
		for _, server := range discoveredHosts {
			var outServer output.OutputBackendServer
			outServer.Host = server.Host
			outServer.Port = server.Port
			outServer.Name = server.Name
			outServer.Disabled = server.Maintenance
			outServer.Weight = server.Weight
			outServer.HAProxyServerOptions = server.HAProxyServerOptions
			backend.Servers = append(backend.Servers,outServer)
		}
		//Get All default servers to include
		if  service.KeepDefaultServers || len(backend.Servers) == 0 {
			for _, server := range service.DefaultServers {
				var outServer output.OutputBackendServer
				outServer.Host = server.Host
				outServer.Port = server.Port
				outServer.Name = server.Name
				outServer.Disabled = false
				outServer.Weight = 0
				backend.Servers = append(backend.Servers,outServer)
			}
		}
		if len(backend.Servers) > 0 {
			sort.Sort(backend.Servers)
			backends = append(backends, backend)
		}
	}
	sort.Sort(backends)
	return backends
}
