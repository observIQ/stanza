package main

import (
	"os"

	"github.com/observiq/stanza/agent"
	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GraphFlags are the flags that can be supplied when running the graph command
type GraphFlags struct {
	*RootFlags
}

// NewGraphCommand creates a command for printing the pipeline as a graph
func NewGraphCommand(rootFlags *RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Args:  cobra.NoArgs,
		Short: "Export a dot-formatted representation of the operator graph",
		Run:   func(command *cobra.Command, args []string) { runGraph(command, args, rootFlags) },
	}
}

func runGraph(_ *cobra.Command, _ []string, flags *RootFlags) {
	var sugaredLogger *zap.SugaredLogger
	if flags.Debug {
		sugaredLogger = newDefaultLoggerAt(zapcore.DebugLevel, "")
	} else {
		sugaredLogger = newDefaultLoggerAt(zapcore.InfoLevel, "")
	}
	defer func() {
		_ = sugaredLogger.Sync()
	}()

	cfg, err := agent.NewConfigFromGlobs(flags.ConfigFiles)
	if err != nil {
		sugaredLogger.Errorw("Failed to read configs from glob", zap.Any("error", err))
		os.Exit(1)
	}

	if err := plugin.RegisterPlugins(flags.PluginDir, operator.DefaultRegistry); err != nil {
		sugaredLogger.Errorw("Failed to register plugins", zap.Any("error", err))
		os.Exit(1)
	}

	buildContext := operator.NewBuildContext(database.NewStubDatabase(), sugaredLogger)
	pipeline, err := cfg.Pipeline.BuildPipeline(buildContext, nil)
	if err != nil {
		sugaredLogger.Errorw("Failed to build operator pipeline", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph, err := pipeline.Render()
	if err != nil {
		sugaredLogger.Errorw("Failed to marshal dot graph", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph = append(dotGraph, '\n')
	_, err = stdout.Write(dotGraph)
	if err != nil {
		sugaredLogger.Errorw("Failed to write dot graph to stdout", zap.Any("error", err))
		os.Exit(1)
	}
}
