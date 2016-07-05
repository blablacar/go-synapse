package output

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type HAProxyOutputSharedFrontend struct {
	Name    string
	Content []string
}

type HAProxyOutput struct {
	Output
	DoWrites               bool
	DoReloads              bool
	DoSocket               bool
	Global                 []string
	Defaults               []string
	ConfigFilePath         string
	ReloadCommandBinary    string
	ReloadCommandArguments []string
	SocketFilePath         string
	WriteInterval          int
	StateFile              string
	StateTTL               int
	BindAddress            string
	waitGroup              sync.WaitGroup
	lastReload             time.Time
	SharedFrontends        []HAProxyOutputSharedFrontend
}

func (h *HAProxyOutput) SetConfiguration(
	DoWrites bool,
	DoReloads bool,
	DoSocket bool,
	Global []string,
	Defaults []string,
	ConfigFilePath string,
	ReloadCommandBinary string,
	ReloadCommandArguments []string,
	SocketFilePath string,
	WriteInterval int,
	StateFile string,
	StateTTL int,
	BindAddress string,
	SharedFrontends []HAProxyOutputSharedFrontend) {

	h.DoWrites = DoWrites
	h.DoReloads = DoReloads
	h.DoSocket = DoSocket
	if len(Global) > 0 {
		h.Global = Global
	}
	if len(Defaults) > 0 {
		h.Defaults = Defaults
	}
	h.ConfigFilePath = ConfigFilePath
	h.ReloadCommandBinary = ReloadCommandBinary
	if len(ReloadCommandArguments) > 0 {
		h.ReloadCommandArguments = ReloadCommandArguments
	}
	h.SocketFilePath = SocketFilePath
	h.WriteInterval = WriteInterval
	h.StateFile = StateFile
	h.StateTTL = StateTTL
	if BindAddress != "" {
		h.BindAddress = BindAddress
	} else {
		h.BindAddress = "localhost"
	}
	h.SharedFrontends = SharedFrontends
}

func (h *HAProxyOutput) isBackendsModified(newBackends OutputBackendSlice) (bool, bool, []string, error) {
	isModified := false
	hasToRestart := false
	var socketCommands []string
	if len(newBackends) == len(h.Backends) {
		for index, backend := range newBackends {
			if len(backend.Servers) != len(h.Backends[index].Servers) {
				isModified = true
				hasToRestart = true
			} else {
				for i, server := range backend.Servers {
					if server.Name == h.Backends[index].Servers[i].Name && server.Host == h.Backends[index].Servers[i].Host && server.Port == h.Backends[index].Servers[i].Port {
						if server.Disabled != h.Backends[index].Servers[i].Disabled {
							isModified = true
							if server.Disabled {
								socketCommands = append(socketCommands, "disable server "+backend.Name+"/"+server.Name)
								log.Debug("disable server " + backend.Name + "/" + server.Name)
							} else {
								socketCommands = append(socketCommands, "enable server "+backend.Name+"/"+server.Name)
								log.Debug("enable server " + backend.Name + "/" + server.Name)
							}
						}
					} else {
						isModified = true
						hasToRestart = true
					}
				}
			}
		}
	} else {
		isModified = true
		hasToRestart = true
	}
	return isModified, hasToRestart, socketCommands, nil
}

//Save the current state of all Backends to StateFile
func (h *HAProxyOutput) SaveState() error {
	data, err := json.Marshal(h.Backends)
	if err != nil {
		log.WithError(err).Warn("Unable to Marchal in JSON Backends State")
		return err
	}
	err = ioutil.WriteFile(h.StateFile, data, 0644)
	if err != nil {
		log.WithField("Filename", h.StateFile).WithError(err).Warn("Unable to Write Backends State into File")
		return err
	}
	return nil
}

//Load the current state of all Backends from StateFile
func (h *HAProxyOutput) LoadState() error {
	if stat, err := os.Stat(h.StateFile); err == nil {
		fileModTime := stat.ModTime()
		now := time.Now()
		expirationDate := fileModTime.Add(time.Duration(h.StateTTL) * time.Millisecond)
		if expirationDate.Before(now) {
			log.Debug("State File exists, but is expired")
			return nil
		}

		// Open and read the configuration file
		file, err := ioutil.ReadFile(h.StateFile)
		if err != nil {
			// If there is an error with opening or reading the configuration file, return the error, and an empty configuration object
			return err
		}

		// Trying to convert the content of the configuration file (theoriticaly in JSON) into a configuration object
		err = json.Unmarshal(file, &h.Backends)
		if err != nil {
			// If there is an error in decoding the JSON entry into configuration object, return a partialy unmarshalled object, and the error
			return err
		}
	} else {
		log.Debug("State File does not exists")
	}

	return nil
}

func (h *HAProxyOutput) modifySharedFrontend(lsfs *[]HAProxyOutputSharedFrontend, Name string, Content []string) {
	for index, sf := range *lsfs {
		if sf.Name == Name {
			for _, str := range Content {
				(*lsfs)[index].Content = append((*lsfs)[index].Content, str)
			}
			return
		}
	}
	//Name not found, create it
	var sf HAProxyOutputSharedFrontend
	sf.Name = Name
	sf.Content = Content
	*lsfs = append(*lsfs, sf)
	return
}

func (h *HAProxyOutput) SaveConfiguration() error {
	if h.DoWrites {
		var lsfs []HAProxyOutputSharedFrontend
		lsfs = h.SharedFrontends
		var data string
		// Write Header
		data = "#\n"
		data += "# HAProxy Configuration File Generated by GO-Synapse\n"
		data += "# If you modify it, be aware that you modifications will be overriden soon\n"
		data += "#\n\n"
		// Global Section
		data += "global\n"
		for _, line := range h.Global {
			data += "  " + line + "\n"
		}
		// Defaults Section
		data += "\ndefaults\n"
		for _, line := range h.Defaults {
			data += "  " + line + "\n"
		}
		data += "\n"
		// Listen Section
		for _, backend := range h.Backends {
			if backend.SharedFrontendName != "" || backend.Port <= 0 {
				data += "backend " + backend.Name + "\n"
				h.modifySharedFrontend(&lsfs, backend.SharedFrontendName, backend.SharedFrontendContent)
				for _, line := range backend.Backend {
					data += "  " + line + "\n"
				}
			} else {
				data += "listen " + backend.Name + "\n"
				data += "  bind " + h.BindAddress + ":" + strconv.Itoa(backend.Port) + "\n"
			}
			for _, line := range backend.Listen {
				data += "  " + line + "\n"
			}
			for _, server := range backend.Servers {
				data += "  server "
				data += server.Name + " "
				data += server.Host + ":"
				data += strconv.Itoa(server.Port) + " "
				data += backend.ServerOptions
				if server.HAProxyServerOptions != "" {
					data += " " + server.HAProxyServerOptions
				}
				if server.Disabled {
					data += " disabled"
				}
				data += "\n"
			}
			data += "\n"
		}
		//Shared Frontend Section
		if len(lsfs) > 0 {
			for _, sf := range lsfs {
				data += "frontend " + sf.Name + "\n"
				for _, str := range sf.Content {
					data += "  " + str + "\n"
				}
			}
			data += "\n"
		}
		err := ioutil.WriteFile(h.ConfigFilePath, []byte(data), 0644)
		if err != nil {
			log.WithField("Filename", h.ConfigFilePath).WithError(err).Warn("Unable to Write HAProxy Configuration File")
			return err
		}
		log.Debug("Configuration File [", h.ConfigFilePath, "] of HAProxy output written")
	} else {
		log.Debug("Do not execute Write modified configuration cause of do_writes flag set to false")
	}
	return nil
}

func (h *HAProxyOutput) reloadHAProxyDaemon() error {
	if h.DoReloads {
		if h.WriteInterval > 0 {
			now := time.Now()
			expirationDate := h.lastReload.Add(time.Duration(h.WriteInterval) * time.Millisecond)
			if expirationDate.After(now) {
				time.Sleep(expirationDate.Sub(now))
			}
			h.lastReload = time.Now()
		}
		var command exec.Cmd
		command.Path = h.ReloadCommandBinary
		command.Args = h.ReloadCommandArguments
		err := command.Run()
		if err != nil {
			log.WithError(err).Warn("HAProxy reloading failed")
			return err
		}
	} else {
		log.Debug("Do not execute restart cause of do_reloads flag set to false")
	}
	return nil
}

func (h *HAProxyOutput) changeBackendsStateBySocket(commands []string) error {
	if h.DoSocket {
		//Send all command to socket
		conn, err := net.Dial("unix", h.SocketFilePath)
		if err != nil {
			log.WithError(err).Warn("Unable to open HAProxy socket to send new backend state")
			return err
		}
		for _, command := range commands {
			_, err = conn.Write([]byte(command))
			if err != nil {
				log.WithError(err).WithField("command", command).Warn("Unable to write command to HAProxy socket")
				conn.Close()
				return err
			}
			buf := make([]byte, 1024)
			n, err := conn.Read(buf[:])
			if err != nil {
				log.WithError(err).WithField("command", command).Warn("Unable to read after command from HAProxy socket")
				conn.Close()
				return err
			}
			return_string := string(buf[0:n])
			if return_string != "\n" {
				log.WithField("command", command).Warn("Unknown error after sending command from HAProxy socket[" + return_string + "]")
				conn.Close()
				return err
			}
		}
		conn.Close()
	} else {
		log.Debug("Do not send modified state to HAProxy cause of do_socket flag set to false")
	}
	return nil
}

func (h *HAProxyOutput) doWork(backends OutputBackendSlice) {
	isModified, hasToRestart, socketCommands, err := h.isBackendsModified(backends)
	if err != nil {
		log.WithError(err).Warn("Error in modification since last check")
		log.Warn("Pretending there's no modification, to keep last valid state informations")
	} else {
		if isModified {
			log.Debug("Backends Configuration modified")
			//Save the new Backend State
			h.Backends = backends
			if h.StateFile != "" {
				h.SaveState()
			}
			//Write the new Configuration file
			err = h.SaveConfiguration()
			if err != nil {
				log.WithError(err).Warn("Unable to Save HAProxy Configuration File")
			} else {
				if hasToRestart {
					//Let's reload the main HAProxy process
					h.reloadHAProxyDaemon()
				} else {
					if h.DoReloads && !h.DoSocket {
						//HAProxy backend state modification by socket forbid by conf
						//So reload the modifications by restarting the daemon
						h.reloadHAProxyDaemon()
					} else {
						//Send command to haproxy using the control socket
						err = h.changeBackendsStateBySocket(socketCommands)
						if err != nil {
							h.reloadHAProxyDaemon()
						}
					}
				}
			}
		} else {
			log.Debug("No modification since last check, nothing to do")
		}
	}
}

func (h *HAProxyOutput) Run(obs_chan chan OutputBackendSlice) {
	log.Debug("Starting HAProxy Run routine")
	defer h.waitGroup.Done()
Loop:
	for {
		select {
		case <-h.Stopper:
			break Loop
		case obs := <-obs_chan:
			h.doWork(obs)
		}
	}
	log.Warn("HAProxy Management Routine stopped")
}

func (h *HAProxyOutput) Initialize() {
	err := h.LoadState()
	if err != nil {
		log.WithError(err).Warn("Unable to load Backends State from file")
		log.Warn("Starting with an empty State")
	}
	h.Stopper = make(chan bool)
	h.waitGroup.Add(1)
}

func (h *HAProxyOutput) Stop() {
	h.Stopper <- true
}

func (h *HAProxyOutput) WaitTermination() {
	h.waitGroup.Wait()
}
