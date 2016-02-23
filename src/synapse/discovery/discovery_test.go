package discovery_test

import (
	"synapse/discovery"
	"testing"
)

func TestCreateDiscovery(t *testing.T) {
	var serviceModified chan bool
	serviceModified = make(chan bool)
	d , err := discovery.CreateDiscovery("base",0,"",nil,serviceModified)
	if err != nil {
		t.Error("Unexpetcted error in Creating Base Discovery: ",err)
	}
	if d.GetType() != "BASE" {
		t.Error("Bad Type for Discovery Expected BASE, got ",d.GetType())
	}
}
