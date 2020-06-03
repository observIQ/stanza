package commands

import (
	"fmt"
	"os"

	"github.com/bluemedora/bplogagent/agent"
	"github.com/bluemedora/bplogagent/plugin"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type GraphFlags struct {
	*RootFlags
}

func NewGraphCommand(rootFlags *RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Args:  cobra.NoArgs,
		Short: "Export a dot-formatted representation of the plugin graph",
		Run:   func(command *cobra.Command, args []string) { runGraph(command, args, rootFlags) },
	}
}

func runGraph(command *cobra.Command, args []string, flags *RootFlags) error {
	var logger *zap.SugaredLogger
	if flags.Debug {
		logger = newDefaultLoggerAt(zapcore.DebugLevel)
	} else {
		logger = newDefaultLoggerAt(zapcore.InfoLevel)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := agent.NewConfigFromGlobs(flags.ConfigFiles)
	if err != nil {
		logger.Errorw("Failed to read configs from glob", zap.Any("error", err))
		os.Exit(1)
	}

	customRegistry, err := plugin.NewCustomRegistry(flags.PluginDir)
	if err != nil {
		logger.Errorw("Failed to load custom plugin registry", zap.Any("error", err))
	}

	buildContext := pg.BuildContext{
		CustomRegistry: customRegistry,
		Logger:         logger,
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
