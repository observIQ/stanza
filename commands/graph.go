package commands

import (
	"fmt"
	"os"

	"github.com/bluemedora/bplogagent/agent"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/custom"
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

	if err := custom.LoadAll(flags.PluginDir, "*.yaml"); err != nil {
		logger.Errorw("Failed to load plugin definitions", zap.Any("error", err))
	}

	cfg, err := agent.NewConfigFromGlobs(flags.ConfigFiles)
	if err != nil {
		logger.Errorw("Failed to read configs from glob", zap.Any("error", err))
		os.Exit(1)
	}

	buildContext := pg.BuildContext{
		Logger: logger,
	}

	pipeline, err := cfg.Pipeline.BuildPipeline(buildContext)
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
