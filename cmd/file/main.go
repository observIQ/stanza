package main

import (
	"os"
	"os/signal"

	pg "github.com/bluemedora/bplogagent/plugin"
	pgs "github.com/bluemedora/bplogagent/plugin/builtin"
	fi "github.com/bluemedora/bplogagent/plugin/builtin/fileinput"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	logCfg := zap.NewDevelopmentConfig()
	logCfg.EncoderConfig.TimeKey = ""
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	baseLogger, _ := logCfg.Build()
	logger := baseLogger.Sugar()

	buildContext := pg.BuildContext{
		Plugins: map[pg.PluginID]pg.Plugin{},
		Logger:  logger,
	}
	logConfig := &pgs.LogOutputConfig{
		DefaultPluginConfig: pg.DefaultPluginConfig{
			PluginID:   "log",
			PluginType: "log",
		},
		Level: "debug",
	}
	logPlugin, err := logConfig.Build(buildContext)
	if err != nil {
		panic(err)
	}

	buildContext.Plugins["log"] = logPlugin
	sourceConfig := &fi.FileSourceConfig{
		DefaultPluginConfig: pg.DefaultPluginConfig{
			PluginID:   "myfilesource",
			PluginType: "file_source",
		},
		DefaultOutputterConfig: pg.DefaultOutputterConfig{
			Output: "log",
		},
		Include: os.Args[1:],
	}

	source, err := sourceConfig.Build(buildContext)
	if err != nil {
		panic(err)
	}

	err = source.(*fi.FileSource).Start()
	if err != nil {
		panic(err)
	}
	logger.Info("Started file source")

	// Wait for interrupt to exit
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	source.(*fi.FileSource).Stop()
}
