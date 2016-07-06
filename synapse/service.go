package synapse

import (
	"encoding/json"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
)

type Service struct {
	Port          int
	Watcher       json.RawMessage
	RouterOptions json.RawMessage

	typedWatcher Watcher
}

func (s *Service) Init() error {
	watcher, err := WatcherFromJson(s.Watcher)
	if err != nil {
		return errs.WithE(err, "Failed to read watcher")
	}
	logs.WithF(watcher.GetFields()).Debug("Watcher loaded")
	s.typedWatcher = watcher
	if err := s.typedWatcher.Init(); err != nil {
		return errs.WithE(err, "Failed to init watcher")
	}
	return nil
}
