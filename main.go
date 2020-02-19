package main

import (
	"time"

	"go.uber.org/zap"
)

type LogCollector struct {
	Config Config

	*zap.SugaredLogger
}

func (l *LogCollector) Start() error {
	l.Info("Starting log collector")
	return nil
}

func (l *LogCollector) Stop() {
	l.Info("Stopping log collector")
}

func (l *LogCollector) Status() struct{} {
	return struct{}{}
}

func main() {
	baseLogger, _ := zap.NewProduction()
	logger := baseLogger.Sugar()

	collector := LogCollector{
		SugaredLogger: logger,
	}

	err := collector.Start()
	if err != nil {
		logger.Errorw("Failed to start log collector", "error", err)
	}

	time.Sleep(1 * time.Second)
	collector.Stop()
}
