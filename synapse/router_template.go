package synapse

import (
	"bufio"
	"bytes"
	"github.com/blablacar/dgr/bin-templater/template"
	"github.com/n0rad/go-erlog/errs"
	"io/ioutil"
	"os"
	"sync"
	"path/filepath"
	"github.com/blablacar/go-nerve/nerve"
)

type RouterTemplate struct {
	RouterCommon
	Template                          string
	TemplateFile                      string
	DestinationFile                   string
	DestinationFileMode               os.FileMode
	PostTemplateCommand               []string
	PostTemplateCommandTimeoutInMilli int

	tmpl                              *template.Templating
}

func NewRouterTemplate() *RouterTemplate {
	return &RouterTemplate{
	}
}

func (r *RouterTemplate) Run(stop chan struct{}, stopWaiter *sync.WaitGroup) {
	r.RunCommon(stop, stopWaiter, r)
}

func (r *RouterTemplate) Init(s *Synapse) error {
	if err := r.commonInit(r, s); err != nil {
		return errs.WithEF(err, r.fields, "Failed to init common router")
	}

	if r.DestinationFile == "" {
		return errs.WithF(r.fields, "DestinationFile is mandatory")
	}
	r.fields = r.fields.WithField("file", r.DestinationFile)
	if r.DestinationFileMode == 0 {
		r.DestinationFileMode = 0644
	}
	if r.Template == "" && r.TemplateFile == "" {
		return errs.WithF(r.fields, "Template or TemplateFile are mandatory")
	}
	if r.Template != "" && r.TemplateFile != "" {
		return errs.WithF(r.fields, "use Template or TemplateFile")
	}
	if r.PostTemplateCommandTimeoutInMilli == 0 {
		r.PostTemplateCommandTimeoutInMilli = 2000
	}

	if r.TemplateFile != "" {
		content, err := ioutil.ReadFile(r.TemplateFile)
		if err != nil {
			return errs.WithEF(err, r.fields.WithField("template", r.TemplateFile), "Failed to read template file")
		}
		r.Template = string(content)
	}

	tmpl, err := template.NewTemplating(nil, r.DestinationFile, r.Template)
	if err != nil {
		return err
	}
	r.tmpl = tmpl
	return nil
}

func (r *RouterTemplate) Update(reports []ServiceReport) error {
	buff := bytes.Buffer{}
	writer := bufio.NewWriter(&buff)
	if err := r.tmpl.Execute(writer, reports); err != nil {
		return errs.WithEF(err, r.fields, "Templating execution failed")
	}

	if err := writer.Flush(); err != nil {
		return errs.WithEF(err, r.fields, "Failed to flush buffer")
	}
	buff.WriteByte('\n')

	if err := os.MkdirAll(filepath.Dir(r.DestinationFile), 0755); err != nil {
		return errs.WithEF(err, r.fields, "Cannot create directories")
	}

	if err := ioutil.WriteFile(r.DestinationFile, buff.Bytes(), r.DestinationFileMode); err != nil {
		return errs.WithEF(err, r.fields, "Failed to write destination file")
	}

	if len(r.PostTemplateCommand) > 0 {
		if err := nerve.ExecCommand(r.PostTemplateCommand, r.PostTemplateCommandTimeoutInMilli); err != nil {
			return errs.WithEF(err, r.fields, "Post template command failed")
		}
	}

	return nil
}

func (r *RouterTemplate) ParseServerOptions(data []byte) (interface{}, error) {
	return nil, nil
}

func (r *RouterTemplate) ParseRouterOptions(data []byte) (interface{}, error) {
	return nil, nil
}
