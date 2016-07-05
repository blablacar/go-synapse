package synapse

import (
	"encoding/json"
	"io/ioutil"
)

type SynapseExtraSectionConfiguration struct {
	Head    string   `json:"head"`
	Content []string `json:"content"`
}

type SynapseCommandHAProxyConfiguration struct {
	Binary    string   `json:"binary"`
	Arguments []string `json:"arguments"`
}

type SynapseSharedFrontendHAProxyConfiguration struct {
	Name    string   `json:"name"`
	Content []string `json:"content"`
}

type SynapseOutputConfiguration struct {
	Type            string                                      `json:"type"`
	OutputFilePath  string                                      `json:"output_file_path"`
	ReloadCommand   SynapseCommandHAProxyConfiguration          `json:"reload_command"`
	ConfigFilePath  string                                      `json:"config_file_path"`
	SocketFilePath  string                                      `json:"socket_file_path"`
	StateFilePath   string                                      `json:"state_file_path"`
	StateFileTTL    int                                         `json:"state_file_ttl"`
	BindAddress     string                                      `json:"bind_address"`
	DoWrites        bool                                        `json:"do_writes"`
	DoReloads       bool                                        `json:"do_reloads"`
	DoSocket        bool                                        `json:"do_socket"`
	Global          []string                                    `json:"global"`
	Defaults        []string                                    `json:"defaults"`
	ExtraSections   []SynapseExtraSectionConfiguration          `json:"extra_sections"`
	RestartInterval int                                         `json:"restart_interval"`
	SharedFrontend  []SynapseSharedFrontendHAProxyConfiguration `json:"shared_frontend"`
}

type SynapseServiceHAProxySharedFrontendConfiguration struct {
	Name    string   `json:"name"`
	Content []string `json:"content"`
}

type SynapseServiceHAProxyConfiguration struct {
	Port           int                                              `json:"port"`
	ServerOptions  string                                           `json:"server_options"`
	Listen         []string                                         `json:"listen"`
	SharedFrontend SynapseServiceHAProxySharedFrontendConfiguration `json:"shared_frontend"`
	Backend        []string                                         `json:"backend"`
}

type SynapseServiceDiscoveryConfiguration struct {
	Type  string   `json:"method"`
	Path  string   `json:"path"`
	Hosts []string `json:"hosts"`
}

type SynapseServiceServerConfiguration struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type SynapseServiceConfiguration struct {
	Name               string                               `json:"name"`
	KeepDefaultServers bool                                 `json:"keep_default_servers"`
	DefaultServers     []SynapseServiceServerConfiguration  `json:"default_servers"`
	Discovery          SynapseServiceDiscoveryConfiguration `json:"discovery"`
	HAProxy            SynapseServiceHAProxyConfiguration   `json:"haproxy"`
}

type SynapseConfiguration struct {
	InstanceID string                        `json:"instance_id"`
	LogLevel   string                        `json:"log-level"`
	Services   []SynapseServiceConfiguration `json:"services"`
	Output     SynapseOutputConfiguration    `json:"output"`
}

// Open Synapse configuration file, and parse it's JSON content
// return a full configuration object and an error
// if the error is different of nil, then the configuration object is empty
// if error is equal to nil, all the JSON content of the configuration file is loaded into the object
func OpenConfiguration(fileName string) (SynapseConfiguration, error) {
	var synapseConfiguration SynapseConfiguration

	// Open and read the configuration file
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		// If there is an error with opening or reading the configuration file, return the error, and an empty configuration object
		return synapseConfiguration, err
	}

	// Trying to convert the content of the configuration file (theoriticaly in JSON) into a configuration object
	err = json.Unmarshal(file, &synapseConfiguration)
	if err != nil {
		// If there is an error in decoding the JSON entry into configuration object, return a partialy unmarshalled object, and the error
		return synapseConfiguration, err
	}

	return synapseConfiguration, nil
}
