package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	// This package registers its HTTP endpoints for profiling using an init hook
	_ "net/http/pprof" // #nosec
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/observiq/stanza/v2/service"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// RootFlags are the root level flags that be provided when invoking stanza from the command line
type RootFlags struct {
	DatabaseFile       string
	ConfigFile         string
	PluginDir          string
	PprofPort          int
	CPUProfile         string
	CPUProfileDuration time.Duration
	MemProfile         string
	MemProfileDelay    time.Duration

	LogConfig string
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
	rootFlagSet.StringVar(&rootFlags.LogConfig, "log_config", "", "specifies the config file to use for logging")
	rootFlagSet.StringVar(&rootFlags.PluginDir, "plugin_dir", defaultPluginDir(), "path to the plugin directory")
	rootFlagSet.StringVar(&rootFlags.DatabaseFile, "database", "", "path to the stanza offset database")

	// Profiling flags
	rootFlagSet.IntVar(&rootFlags.PprofPort, "pprof_port", 0, "listen port for pprof profiling")
	rootFlagSet.StringVar(&rootFlags.CPUProfile, "cpu_profile", "", "path to cpu profile output")
	rootFlagSet.DurationVar(&rootFlags.CPUProfileDuration, "cpu_profile_duration", 60*time.Second, "duration to run the cpu profile")
	rootFlagSet.StringVar(&rootFlags.MemProfile, "mem_profile", "", "path to memory profile output")
	rootFlagSet.DurationVar(&rootFlags.MemProfileDelay, "mem_profile_delay", 10*time.Second, "time to wait before writing a memory profile")

	// Set profiling flags to hidden
	hiddenFlags := []string{"pprof_port", "cpu_profile", "cpu_profile_duration", "mem_profile", "mem_profile_delay"}
	for _, flag := range hiddenFlags {
		err := rootFlagSet.MarkHidden(flag)
		if err != nil {
			// MarkHidden only fails if the flag does not exist
			panic(err)
		}
	}

	root.AddCommand(NewGraphCommand(rootFlags))
	root.AddCommand(NewVersionCommand())
	root.AddCommand(NewOffsetsCmd(rootFlags))

	return root
}

func runRoot(command *cobra.Command, _ []string, flags *RootFlags) {
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

	logger := service.NewLogger(*conf.Logging).Sugar()
	defer func() {
		_ = logger.Sync()
	}()

	// Build agent service
	service, ctx, err := service.NewBuilder().
		WithConfigFile(conf).
		WithDatabaseFile(flags.DatabaseFile).
		WithLogger(logger).
		Build(command.Context())
	if err != nil {
		logger.Errorf("Failed to create agent service", zap.Any("error", err))
		os.Exit(1)
	}

	profilingWg := startProfiling(ctx, flags, logger)

	err = service.Run()
	if err != nil {
		logger.Errorw("Failed to run agent service", zap.Any("error", err))
		os.Exit(1)
	}

	profilingWg.Wait()
}

func startProfiling(ctx context.Context, flags *RootFlags, logger *zap.SugaredLogger) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	// Start pprof listening on port
	if flags.PprofPort != 0 {
		// pprof endpoints registered by importing net/pprof
		var srv http.Server
		srv.Addr = fmt.Sprintf(":%d", flags.PprofPort)

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info(srv.ListenAndServe())
		}()

		wg.Add(1)

		go func() {
			defer wg.Done()
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := srv.Shutdown(shutdownCtx)
			if err != nil {
				logger.Warnw("Errored shutting down pprof server", zap.Error(err))
			}
		}()
	}

	// Start CPU profile for configured duration
	if flags.CPUProfile != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			f, err := os.Create(flags.CPUProfile)
			if err != nil {
				logger.Errorw("Failed to create CPU profile", zap.Error(err))
				return
			}

			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal("could not start CPU profile: ", err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(flags.CPUProfileDuration):
			}
			pprof.StopCPUProfile()

			if f != nil {
				if err := f.Close(); err != nil { // #nosec G307
					logger.Errorf(err.Error())
				}
			}
		}()
	}

	// Start memory profile after configured delay
	if flags.MemProfile != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
			case <-time.After(flags.MemProfileDelay):
			}

			f, err := os.Create(flags.MemProfile)
			if err != nil {
				logger.Errorw("Failed to create memory profile", zap.Error(err))
			}

			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}

			if f != nil {
				if err := f.Close(); err != nil {
					logger.Errorw("Failed to close file", zap.Error(err))
				}
			}
		}()
	}

	return wg
}
