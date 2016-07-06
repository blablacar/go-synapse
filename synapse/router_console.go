package synapse

import (
	"encoding/json"
	"github.com/blablacar/go-nerve/nerve"
	"github.com/n0rad/go-erlog/errs"
	"io"
	"os"
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

func (r *RouterConsole) Init() error {
	if err := r.commonInit(); err != nil {
		return errs.WithEF(err, r.fields, "Failed to init common router")
	}
	return nil
}

func (r *RouterConsole) Update(backends []nerve.Report) error {
	res, err := json.Marshal(backends)
	if err != nil {
		return errs.WithEF(err, r.fields, "Failed to prepare router update")
	}
	println(res)
	return nil
}
