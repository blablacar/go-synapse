package synapse

import (
	"encoding/json"
	"fmt"
	"github.com/n0rad/go-erlog/errs"
	"io"
	"os"
	"sync"
)

type RouterConsole struct {
	RouterCommon

	writer io.Writer
}

func NewRouterConsole() *RouterConsole {
	return &RouterConsole{
		writer: os.Stdout,
	}
}

func (r *RouterConsole) Run(stop chan struct{}, stopWaiter *sync.WaitGroup) {
	r.RunCommon(stop, stopWaiter, r)
}

func (r *RouterConsole) Init(s *Synapse) error {
	if err := r.commonInit(r, s); err != nil {
		return errs.WithEF(err, r.fields, "Failed to init common router")
	}
	return nil
}

func (r *RouterConsole) Update(serviceReport ServiceReport) error {
	res, err := json.Marshal(serviceReport.reports)
	if err != nil {
		return errs.WithEF(err, r.fields, "Failed to prepare router update")
	}
	fmt.Fprintf(r.writer, "%s\n", res)
	return nil
}

func (r *RouterConsole) ParseServerOptions(data []byte) (interface{}, error) {
	return nil, nil
}

func (r *RouterConsole) ParseRouterOptions(data []byte) (interface{}, error) {
	return nil, nil
}
