package main

import (
	"time"

	bpla "github.com/bluemedora/bplogagent"
	"go.uber.org/zap"
)

func main() {
	baseLogger, _ := zap.NewProduction()
	logger := baseLogger.Sugar()

	collector := bpla.LogAgent{
		SugaredLogger: logger,
	}

	err := collector.Start()
	if err != nil {
		logger.Errorw("Failed to start log collector", "error", err)
	}

	time.Sleep(1 * time.Second)
	collector.Stop()
}
