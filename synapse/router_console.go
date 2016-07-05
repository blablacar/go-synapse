package synapse

import (
	"github.com/blablacar/go-nerve/nerve"
	"os"
	"io"
	"github.com/n0rad/go-erlog/errs"
)

type RouterConsole struct {
	RouterCommon

	writer io.Writer
}

func NewRouterFile() *RouterConsole {
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

func (r *RouterConsole) Start(stop chan struct{}) {
	//data, err := json.Marshal(backends)
	//if err != nil {
	//	return errs.WithEF(err, r.fields, "Unable to marshal backends")
	//}
	//if err := ioutil.WriteFile(r.FilePath, data, 0644); err != nil {
	//	return errs.WithEF(err, r.fields.WithField("file", r.FilePath), "Failed to write to router file")
	//}
	//return nil
}


func (r *RouterConsole) Run(backends []nerve.Report) error {
	return nil
}
