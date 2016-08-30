package synapse

import (
	"context"
	"sync"
)

//type Context interface {
//	Done() <-chan struct{}
//	Err() error
//}
//
//type TimeoutContext interface {
//	Context
//	Deadline() time.Time
//}

type SynapseContext interface {
	context.Context
	oneshot() bool
}

func NewSynapseContext(parent SynapseContext, oneshot bool) SynapseContext {
	return &SynapseContextImpl{
		parent,
		//oneshot,
	}
}

type SynapseContextImpl struct {
	SynapseContext
}

type ContextImpl struct {
	stop       chan struct{} //TODO this should not be bidirectionnal
	doneWaiter *sync.WaitGroup
	oneshot    bool
}

func newContext(oneshot bool) *ContextImpl {
	return &ContextImpl{
		stop:       make(chan struct{}),
		doneWaiter: &sync.WaitGroup{},
		oneshot:    oneshot,
	}
}
