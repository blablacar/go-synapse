package output

import (
	log "github.com/Sirupsen/logrus"
	"encoding/json"
	"io/ioutil"
	"sync"
)

type FileOutput struct {
	Output
	FilePath string
	WriteInterval int
	waitGroup sync.WaitGroup
}

func(f *FileOutput) SetConfiguration(FilePath string,WriteInterval int) {
	f.FilePath = FilePath
	f.WriteInterval = WriteInterval
}

//Save the current state of all Backends to StateFile
func(f *FileOutput) WriteBackendsState() error {
	data, err := json.Marshal(f.Backends)
	if err != nil {
		log.WithError(err).Warn("Unable to Marchal in JSON Backends State")
		return err
	}
	err = ioutil.WriteFile(f.FilePath,data,0644)
	if err != nil {
		log.WithField("Filename",f.FilePath).WithError(err).Warn("Unable to Write Backends State into File")
		return err
	}
	return nil
}

func(f *FileOutput) Run(obs_chan chan OutputBackendSlice) {
	defer f.waitGroup.Done()
	Loop:
	for {
		select {
			case obs := <-obs_chan:
				f.Backends = obs
				f.WriteBackendsState()
			case <-f.Stopper:
				log.Debug("Close Signal Receive in File Output Routine")
				break Loop
		}
	}
}

func(f *FileOutput) Initialize() {
	f.Stopper = make(chan bool)
	f.waitGroup.Add(1)
}

func(f *FileOutput) Stop() {
	f.Stopper <- true
}

func(f *FileOutput) WaitTermination() {
	f.waitGroup.Wait()
}
