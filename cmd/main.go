package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"

	"net/http"
	_ "net/http/pprof"

	bpla "github.com/bluemedora/bplogagent"
	"github.com/bluemedora/bplogagent/config"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func main() {
	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logCfg.Sampling.Initial = 5
	logCfg.Sampling.Thereafter = 100
	logCfg.EncoderConfig.CallerKey = ""
	// logCfg := zap.NewDevelopmentConfig()
	// logCfg.EncoderConfig.TimeKey = ""
	// logCfg.EncoderConfig.CallerKey = ""
	// logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	baseLogger, _ := logCfg.Build()
	logger := baseLogger.Sugar()
	defer func() {
		_ = logger.Sync()
	}()

	var cfg config.Config
	var configFile string
	flag.StringVar(&configFile, "config", "/etc/bplogagent/config.yml", "Path to the config file")
	flag.StringVar(&cfg.PluginGraphOutput, "graph", "", "Path to output a dot formatted representation of the plugin graph")
	flag.Parse()

	configContents, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Errorw("Failed to read config file", zap.Error(err))
		return
	}

	err = yaml.Unmarshal(configContents, &cfg)
	if err != nil {
		logger.Errorw("Failed to parse config file", zap.Error(err))
		return
	}

	logger.Debugw("Parsed config", "config", cfg)

	agent := bpla.NewLogAgent(&cfg, logger)

	err = agent.Start()
	if err != nil {
		logger.Errorw("Failed to start log agent", zap.Any("error", err))
		return
	}

	// Start the profiler http server
	go func() {
		logger.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	// Wait for interrupt to exit
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	agent.Stop()
}
