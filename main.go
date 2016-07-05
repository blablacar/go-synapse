package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/blablacar/go-synapse/synpase"
	"os"
	"os/signal"
	"time"
)

var (
	Version   = "No Version Defined"
	BuildTime = "1970-01-01_00:00:00_UTC"
)

// Manage OS Signal, only for shutdown purpose
// When termination signal is received, we send a message to a chan
func manageSignal(c <-chan os.Signal, stop chan<- bool) {
	for {
		select {
		case _signal := <-c:
			if _signal == os.Kill {
				log.Debug("Synapse: Kill Signal Received")
			}
			if _signal == os.Interrupt {
				log.Debug("Synapse: Interrupt Signal Received")
			}
			log.Info("Synapse: Shutdown Signal Received")
			stop <- true
			break
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// Set the Logrus global log level
// Converted from a configuration string
func setLogLevel(logLevel string) {
	// Set the Log Level, extracted from the command line
	switch logLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}
}

func printVersion() {
	fmt.Println("Synapse")
	fmt.Println("Version :", Version)
	fmt.Println("Build Time :", BuildTime)
}

// All the command line arguments are managed inside this function
func initFlags() (string, string) {
	// The Log Level, from the Sirupsen/logrus level
	var logLevel = flag.String("log-level", "WARN", "A value to choose between [DEBUG INFO WARN FATAL], can be overriden by config file")
	// The configuration filename
	var configurationFileName = flag.String("config", "./synapse.json.conf", "the complete filename of the configuration file")
	// The version flag
	var version = flag.Bool("version", false, "Display version and exit")
	// Parse all command line options
	flag.Parse()
	if *version {
		printVersion()
		os.Exit(0)
	}

	return *logLevel, *configurationFileName
}

func initConfiguration(configurationFileName string) synapse.SynapseConfiguration {
	log.Debug("Synapse: Starting config file parsing")
	synapseConfiguration, err := synapse.OpenConfiguration(configurationFileName)
	if err != nil {
		// If an error is raised when parsing configuration file
		// the configuration object can be either empty, either incomplete
		log.WithError(err).Fatal("Synapse: Unable to load Configuration")
		// So the configuration is incomplete, exit the program now
		os.Exit(1)
	}
	log.Debug("Synapse: Config file parsed")
	return synapseConfiguration
}

func main() {
	// Init flags, to get logLevel and configuration file name
	logLevel, configurationFileName := initFlags()

	setLogLevel(logLevel)

	// Load the configuration from config file. If something wrong, the full process is stopped inside the function
	synapseConfiguration := initConfiguration(configurationFileName)

	// If the log level is setted inside the configuration file, override the command line level
	if synapseConfiguration.LogLevel != "" {
		setLogLevel(synapseConfiguration.LogLevel)
	}

	log.Info("Synapse: Starting Run of Instance")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	log.Debug("Synapse: Signal Channel notification setup done")

	stop := make(chan bool)
	go manageSignal(c, stop)
	log.Debug("Synapse: Signal Management Started")

	finished := make(chan bool)
	go synapse.Run(stop, finished, synapseConfiguration)
	log.Debug("Synapse: Go routine launched")

	log.Debug("Synapse: Waiting for main process to Stop")
	isFinished := <-finished
	if isFinished {
		log.Debug("Synapse: Main routine closed correctly")
	} else {
		log.Warn("Synapse: Main routine closed incorrectly")
	}
	log.Info("Synapse: Shutdown of Instance Completed")
}
