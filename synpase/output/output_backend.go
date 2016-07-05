package output

type OutputBackendServer struct {
	Name                 string
	Host                 string
	Port                 int
	Disabled             bool
	Weight               int
	HAProxyServerOptions string
	Tags                 []string
}

type OutputBackendSlice []OutputBackend

type OutputBackend struct {
	Name                  string
	Port                  int
	ServerOptions         string
	Listen                []string
	Backend               []string
	Servers               OutputBackendServerSlice
	SharedFrontendName    string
	SharedFrontendContent []string
}

type OutputBackendServerSlice []OutputBackendServer

func (obs OutputBackendSlice) Len() int {
	return len(obs)
}

func (obs OutputBackendSlice) Swap(i, j int) {
	obs[i], obs[j] = obs[j], obs[i]
}

func (obs OutputBackendSlice) Less(i, j int) bool {
	return obs[i].Name < obs[j].Name
}

func (obss OutputBackendServerSlice) Len() int {
	return len(obss)
}

func (obss OutputBackendServerSlice) Swap(i, j int) {
	obss[i], obss[j] = obss[j], obss[i]
}

func (obss OutputBackendServerSlice) Less(i, j int) bool {
	return obss[i].Name < obss[j].Name
}
