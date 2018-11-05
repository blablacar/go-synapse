package synapse

import (
	"testing"

	"github.com/blablacar/go-nerve/nerve"
)

func TestIsSocketUpdatable(t *testing.T) {
	s := &Synapse{}
	s.Init("version", "buildtime", true)

	r := NewRouterHaProxy()
	r.Global = []string{"stats socket /tmp/haproxy.sock"}
	r.Init(s)

	yes := true
	no := false

	NodeA := Report{
		nerve.Report{
			Available:            &yes,
			Host:                 "10.0.0.1",
			Port:                 8080,
			Name:                 "NodeA",
			HaProxyServerOptions: "",
		},
		int64(0),
	}
	NodeB := Report{
		nerve.Report{
			Available:            &no,
			Host:                 "10.0.0.1",
			Port:                 8080,
			Name:                 "NodeB",
			HaProxyServerOptions: "",
		},
		int64(0),
	}

	srNew := ServiceReport{
		Service: &Service{
			Name: "ServiceA",
		},
		Reports: []Report{
			NodeA,
		},
	}

	r.lastEvents = map[string]*ServiceReport{
		"ServiceA": {
			Service: &Service{
				Name: "ServiceA",
			},
			Reports: []Report{
				NodeA,
			},
		},
	}

	// last has nodeA, new has nodeA
	if u := r.isSocketUpdatable(srNew); !u {
		t.Errorf("isSocketUpdatable should be true, was %v", u)
	}

	// last has nodeA, new has nodeA+nodeB
	srNew.Reports = append(srNew.Reports, NodeB)
	if u := r.isSocketUpdatable(srNew); u {
		t.Errorf("isSocketUpdatable should be false, was %v", u)
	}

	// last has nodeA+nodeB, new has nodeA+nodeB
	r.lastEvents["ServiceA"].Reports = append(r.lastEvents["ServiceA"].Reports, NodeB)
	if u := r.isSocketUpdatable(srNew); !u {
		t.Errorf("isSocketUpdatable should be true, was %v", u)
	}
}
