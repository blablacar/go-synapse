package output

type Output struct {
	Backends OutputBackendSlice
	StateFile string
	WriteInterval int
	backendsChan chan OutputBackendSlice
	Stopper chan bool
}

type OutputI interface {
	Run(chan OutputBackendSlice)
	Stop()
	WaitTermination()
	Initialize()
	SetBackends(OutputBackendSlice)
}

func CreateOutput(
	Type string,
	FilePath string,
	DoWrites bool,
        DoReloads bool,
        DoSocket bool,
        Global []string,
        Defaults []string,
        ReloadCommandBinary string,
        ReloadCommandArguments []string,
        SocketFilePath string,
        WriteInterval int,
        StateFile string,
        StateTTL int,
	BindAddress string) OutputI {

	var returnOutput OutputI
	switch(Type) {
	case "haproxy":
		var output HAProxyOutput
		output.SetConfiguration(
			DoWrites,
			DoReloads,
			DoSocket,
			Global,
			Defaults,
			FilePath,
			ReloadCommandBinary,
			ReloadCommandArguments,
			SocketFilePath,
			WriteInterval,
			StateFile,
			StateTTL,
			BindAddress)
		returnOutput = &output
	case "file":
		var output FileOutput
		output.SetConfiguration(FilePath,WriteInterval)
		returnOutput = &output
	}
	return returnOutput
}

func(o *Output) SetBackends(backends OutputBackendSlice) {
	o.Backends = backends
}
