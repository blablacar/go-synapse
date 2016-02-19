package synapse_test

import (
	"synapse"
	"testing"
)

func TestGetAllBackends(t *testing.T) {
	var testableSO synapse.SynapseOutput;
	oBS := testableSO.GetAllBackends()
	if len(oBS) != 0 {
		t.Error("Expected 0 backends to be loaded, got ",len(oBS))
	}
}
