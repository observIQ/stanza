package main

import (
	"bytes"
	"time"

	bpla "github.com/bluemedora/bplogagent"
	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	baseLogger, _ := zap.NewDevelopment()
	logger := baseLogger.Sugar()

	rawConfig := []byte(`
plugins:
- id: fdsa
  type: generate
  interval: 1
  output: myjson
  record:
    test: asdf
- id: myjson
  type: json
  output: mylogger
- id: mylogger
  type: logger
`)

	var cfg config.Config
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewReader(rawConfig))
	if err != nil {
		logger.Errorw("Failed to read the config", "error", err)
		return
	}
	err = v.Unmarshal(&cfg, plugin.UnmarshalHook)
	if err != nil {
		logger.Errorw("Failed to unmarshal the config", "error", err)
		return
	}

	logger.Infow("Unmarshalled the config", "config", cfg)

	agent := bpla.LogAgent{
		SugaredLogger: logger,
		Config:        cfg,
	}

	err = agent.Start()
	if err != nil {
		logger.Errorw("Failed to start log collector", "error", err)
		return
	}

	time.Sleep(1 * time.Second)
	agent.Stop()
}
