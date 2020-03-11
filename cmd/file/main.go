package main

import (
	"os"
	"os/signal"

	pg "github.com/bluemedora/bplogagent/plugin"
	pgs "github.com/bluemedora/bplogagent/plugin/plugins"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	out := &pgs.DropOutput{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:   "drop",
			PluginType: "drop",
		},
	}

	logCfg := zap.NewDevelopmentConfig()
	logCfg.EncoderConfig.TimeKey = ""
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	baseLogger, _ := logCfg.Build()
	logger := baseLogger.Sugar()
	sourceConfig := &pgs.FileSourceConfig{
		DefaultPluginConfig: pg.DefaultPluginConfig{
			PluginID:   "myfilesource",
			PluginType: "file_source",
		},
		DefaultOutputterConfig: pg.DefaultOutputterConfig{
			Output: "drop",
		},
		Include: os.Args[1:],
	}
	buildContext := pg.BuildContext{
		Plugins: map[pg.PluginID]pg.Plugin{
			pg.PluginID("drop"): out,
		},
		Logger: logger,
	}

	source, err := sourceConfig.Build(buildContext)
	if err != nil {
		panic(err)
	}

	err = source.(*pgs.FileSource).Start()
	if err != nil {
		panic(err)
	}
	logger.Info("Started file source")

	// Wait for interrupt to exit
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	source.(*pgs.FileSource).Stop()
}
