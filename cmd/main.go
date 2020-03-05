package main

import (
	"flag"
	"os"
	"os/signal"

	bpla "github.com/bluemedora/bplogagent"
	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	logCfg := zap.NewDevelopmentConfig()
	logCfg.EncoderConfig.TimeKey = ""
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	baseLogger, _ := logCfg.Build()
	logger := baseLogger.Sugar()
	defer func() {
		_ = logger.Sync()
	}()

	var configFile string
	flag.StringVar(&configFile, "config", "/etc/bplogagent/config.yml", "Path to the config file")
	flag.Parse()

	var cfg config.Config
	v := viper.New()
	v.SetConfigFile(configFile)
	err := v.ReadInConfig()
	if err != nil {
		logger.Errorw("Failed to read the config", "error", err)
		return
	}
	err = v.Unmarshal(&cfg, plugin.UnmarshalHook)
	if err != nil {
		logger.Errorw("Failed to unmarshal the config", "error", err)
		return
	}

	cfgYaml, err := yaml.Marshal(cfg)
	if err != nil {
		logger.Errorw("Failed to marshal yaml", "error", err)
	}
	logger.Infof("Unmarshalled the config: %s\n", string(cfgYaml))

	agent := bpla.NewLogAgent(cfg, logger)

	err = agent.Start()
	if err != nil {
		logger.Errorw("Failed to start log collector", "error", err)
		return
	}

	// Wait for interrupt to exit
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	agent.Stop()
}
