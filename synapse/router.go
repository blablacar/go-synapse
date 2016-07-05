package synapse

import (
	"github.com/n0rad/go-erlog/errs"
	"encoding/json"
	"github.com/n0rad/go-erlog/data"
)

type RouterCommon struct {
	Type   string
	Services []Service

	fields data.Fields
}

type Router interface {
	Init() error
	getFields() data.Fields
	Start(stop chan struct{})
}

func (r *RouterCommon) commonInit() error {
	r.fields = data.WithField("type", r.Type)
	for _, service := range r.Services {
		if err := service.init(); err != nil {
			return errs.WithEF(err, r.fields, "Failed to init service")
		}
	}

	return nil
}

func (r *RouterCommon) getFields() data.Fields {
	return r.fields
}

func RouterFromJson(content []byte) (Router, error) {
	t := &RouterCommon{}
	if err := json.Unmarshal([]byte(content), t); err != nil {
		return nil, errs.WithE(err, "Failed to unmarshall check type")
	}

	fields := data.WithField("type", t.Type)
	var typedRouter Router
	switch t.Type {
	case "file":
		typedRouter = NewRouterFile()
	default:
		return nil, errs.WithF(fields, "Unsupported router type")
	}

	if err := json.Unmarshal([]byte(content), &typedRouter); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to unmarshall router")
	}

	if err := typedRouter.Init(); err != nil {
		return nil, errs.WithEF(err, fields, "Failed to init router")
	}
	return typedRouter, nil
}