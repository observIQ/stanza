package commands

import (
	"fmt"
	"os"

	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/pipeline"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type GraphFlags struct {
	*RootFlags
}

func runGraph(command *cobra.Command, args []string, flags *GraphFlags) error {
	var logger *zap.SugaredLogger
	if flags.Debug {
		logger = newDefaultLoggerAt(zapcore.DebugLevel)
	} else {
		logger = newDefaultLoggerAt(zapcore.InfoLevel)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.ReadConfigsFromGlobs(flags.ConfigFiles)
	if err != nil {
		logger.Errorw("Failed to read config files", zap.Any("error", err))
		os.Exit(1)
	}

	buildContext := pg.BuildContext{
		Logger: logger,
	}
	plugins, err := pg.BuildPlugins(cfg.Plugins, buildContext)
	if err != nil {
		logger.Errorw("Failed to build plugins", zap.Any("error", err))
		os.Exit(1)
	}

	pipeline, err := pipeline.NewPipeline(plugins)
	if err != nil {
		logger.Errorw("Failed to build plugin pipeline", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph, err := pipeline.MarshalDot()
	if err != nil {
		logger.Errorw("Failed to marshal dot graph", zap.Any("error", err))
		os.Exit(1)
		return fmt.Errorf("marshal dot graph: %s", err)
	}

	_, err = os.Stdout.Write(dotGraph)
	if err != nil {
		logger.Errorw("Failed to write dot graph to stdout", zap.Any("error", err))
		os.Exit(1)
		return fmt.Errorf("write dot graph to stdout: %s", err)
	}

	return nil
}
