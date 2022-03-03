package main

import (
	"log"

	// This package registers its HTTP endpoints for profiling using an init hook
	_ "net/http/pprof" // #nosec
	"os"

	"github.com/observiq/stanza/v2/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	envStanzaLogFile      = "STANZA_LOG_FILE"
	envStanzaDatabaseFile = "STANZA_DATABASE_FILE"
)

// RootFlags are the root level flags that be provided when invoking stanza from the command line
type RootFlags struct {
	DatabaseFile string
	ConfigFile   string
	PluginDir    string
}

// NewRootCmd will return a root level command
func NewRootCmd() *cobra.Command {
	rootFlags := &RootFlags{}

	root := &cobra.Command{
		Use:   "stanza [-c ./config.yaml]",
		Short: "A log parser and router",
		Long:  "A log parser and router",
		Args:  cobra.NoArgs,
		Run:   func(command *cobra.Command, args []string) { runRoot(command, args, rootFlags) },
	}

	rootFlagSet := root.PersistentFlags()
	rootFlagSet.StringVarP(&rootFlags.ConfigFile, "config", "c", defaultConfig(), "path to a config file")
	rootFlagSet.StringVar(&rootFlags.PluginDir, "plugin_dir", defaultPluginDir(), "path to the plugin directory")
	rootFlagSet.StringVar(&rootFlags.DatabaseFile, "database", "", "path to the stanza offset database")

	root.AddCommand(NewGraphCommand(rootFlags))
	root.AddCommand(NewVersionCommand())
	root.AddCommand(NewOffsetsCmd(rootFlags))

	return root
}

func runRoot(command *cobra.Command, _ []string, flags *RootFlags) {
	conf, err := service.LoadConfig(flags.PluginDir, flags.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	logger := service.NewLogger(*conf.Logging).Sugar()
	defer func() {
		_ = logger.Sync()
	}()

	// Build agent service
	service, err := service.NewBuilder().
		WithConfig(conf).
		WithDatabaseFile(flags.DatabaseFile).
		WithLogger(logger).
		Build(command.Context())
	if err != nil {
		logger.Errorw("Failed to create agent service", zap.Error(err))
		os.Exit(1)
	}

	err = service.Run()
	if err != nil {
		logger.Errorw("Failed to run agent service", zap.Error(err))
		os.Exit(1)
	}
}
