package bplogagent

import (
	"go.uber.org/zap"
)

type LogAgent struct {
	Config Config

	*zap.SugaredLogger
}

func (l *LogAgent) Start() error {
	l.Info("Starting log collector")
	return nil
}

func (l *LogAgent) Stop() {
	l.Info("Stopping log collector")
}

func (l *LogAgent) Status() struct{} {
	return struct{}{}
}
