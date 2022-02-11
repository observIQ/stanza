package main

import (
	"log"
	"os"

	"github.com/observiq/stanza/v2/service"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	if flags.PluginDir != "" {
		// Plugins MUST be loaded before calling LoadConfig, otherwise stanza will fail to recognize plugin
		// types, and fail to load any config using plugins
		if errs := plugin.RegisterPlugins(flags.PluginDir, operator.DefaultRegistry); len(errs) != 0 {
			log.Fatalf("Got errors parsing plugins %s", errs)
		}
	}

	conf, err := service.LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	err = conf.Logging.Validate()
	if err != nil {
		log.Fatalf("Failed to validate logging config: %s", err.Error())
	}

	logger := service.NewLogger(*conf.Logging).Sugar()
	defer func() {
		_ = logger.Sync()
	}()

	buildContext := operator.NewBuildContext(logger)
	pipeline, err := conf.Pipeline.BuildPipeline(buildContext, nil)
	if err != nil {
		logger.Errorw("Failed to build operator pipeline", zap.Any("error", err))
		os.Exit(1)
	}

	dotGraph, err := pipeline.Render()
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
