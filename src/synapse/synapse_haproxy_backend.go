package synapse

type HAProxyBackendServer struct {
	Name string
	Host string
	Port int
	Disabled bool
	Weight int
	HAProxyServerOptions string
	Tags []string
}

type HAProxyBackend struct {
	Name string
	Port int
	ServerOptions string
	Listen []string
	Servers HAProxyBackendServerSlice
}

type HAProxyBackendSlice []HAProxyBackend
type HAProxyBackendServerSlice []HAProxyBackendServer

func(h HAProxyBackendSlice) Len() int{
	return len(h)
}

func(h HAProxyBackendSlice) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func(h HAProxyBackendSlice) Less(i, j int) bool{
	return h[i].Name < h[j].Name
}

func(hs HAProxyBackendServerSlice) Len() int{
	return len(hs)
}

func(hs HAProxyBackendServerSlice) Swap(i, j int) {
	hs[i], hs[j] = hs[j], hs[i]
}

func(hs HAProxyBackendServerSlice) Less(i, j int) bool {
	return hs[i].Name < hs[j].Name
}
