package synapse_test

import (
	"synapse"
	"testing"
)

func TestOpenConfiguration(t *testing.T) {
	config, err := synapse.OpenConfiguration("../../example/synapse.conf.json")
	if err != nil {
		t.Fatal("Unable to open Configuration file", err)
	}
	if config.InstanceID != "mymachine" {
		t.Error("Expected instance_id to be 'mymachine', got ",config.InstanceID)
	}
	if config.LogLevel != "DEBUG" {
		t.Error("Expected log-level to be 'DEBUG', got ",config.LogLevel)
	}
	if len(config.Services) != 2 {
		t.Error("Expected 2 services to be loaded, got ",len(config.Services))
	}
}
