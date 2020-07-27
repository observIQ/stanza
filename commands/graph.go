package commands

import (
	"os"

	"github.com/observiq/carbon/agent"
	"github.com/observiq/carbon/operator"
	pg "github.com/observiq/carbon/operator"
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
	var logger *zap.SugaredLogger
	if flags.Debug {
		logger = newDefaultLoggerAt(zapcore.DebugLevel, "")
	} else {
		logger = newDefaultLoggerAt(zapcore.InfoLevel, "")
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := agent.NewConfigFromGlobs(flags.ConfigFiles)
	if err != nil {
		logger.Errorw("Failed to read configs from glob", zap.Any("error", err))
		os.Exit(1)
	}

	pluginRegistry, err := operator.NewPluginRegistry(flags.PluginDir)
	if err != nil {
		logger.Errorw("Failed to load plugin registry", zap.Any("error", err))
	}

	buildContext := pg.BuildContext{
		PluginRegistry: pluginRegistry,
		Logger:         logger,
	}

	pipeline, err := cfg.Pipeline.BuildPipeline(buildContext)
	if err != nil {
		logger.Errorw("Failed to build operator pipeline", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph, err := pipeline.MarshalDot()
	if err != nil {
		logger.Errorw("Failed to marshal dot graph", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph = append(dotGraph, '\n')
	_, err = stdout.Write(dotGraph)
	if err != nil {
		logger.Errorw("Failed to write dot graph to stdout", zap.Any("error", err))
		os.Exit(1)
	}
}
