package synapse_test

import (
	"github.com/blablacar/go-synapse/synpase"
	"testing"
)

func TestGetAllBackends(t *testing.T) {
	var testableSO synapse.SynapseOutput
	oBS := testableSO.GetAllBackends()
	if len(oBS) != 0 {
		t.Error("Expected 0 backends to be loaded, got ", len(oBS))
	}
}
