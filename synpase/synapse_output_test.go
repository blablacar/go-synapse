package synapse

import (
	"testing"
)

func TestGetAllBackends(t *testing.T) {
	var testableSO SynapseOutput
	oBS := testableSO.GetAllBackends()
	if len(oBS) != 0 {
		t.Error("Expected 0 backends to be loaded, got ", len(oBS))
	}
}
