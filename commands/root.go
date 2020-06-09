package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	agent "github.com/bluemedora/bplogagent/agent"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RootFlags struct {
	DatabaseFile       string
	ConfigFiles        []string
	PluginDir          string
	PprofPort          int
	CPUProfile         string
	CPUProfileDuration time.Duration
	MemProfile         string
	MemProfileDelay    time.Duration
	Debug              bool
}

func NewRootCmd() *cobra.Command {
	rootFlags := &RootFlags{}

	root := &cobra.Command{
		Use:   "bplogagent [-c ./config.yaml]",
		Short: "A log parser and router",
		Long:  "A log parser and router",
		Args:  cobra.NoArgs,
		Run:   func(command *cobra.Command, args []string) { runRoot(command, args, rootFlags) },
	}

	rootFlagSet := root.PersistentFlags()
	rootFlagSet.StringSliceVarP(&rootFlags.ConfigFiles, "config", "c", []string{"./config.yaml"}, "path to a config file") // TODO default locations
	rootFlagSet.StringVar(&rootFlags.PluginDir, "plugin_dir", "./plugins", "path to the plugin directory")
	rootFlagSet.StringVar(&rootFlags.DatabaseFile, "database", "./bplogagent.db", "path to the log agent offset database")
	rootFlagSet.BoolVar(&rootFlags.Debug, "debug", false, "debug logging")

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
		logger.Errorw("Failed to read configs from globs", zap.Any("error", err), zap.Any("globs", flags.ConfigFiles))
		os.Exit(1)
	}
	logger.Debugw("Parsed config", "config", cfg)
	cfg.SetDefaults(flags.DatabaseFile, flags.PluginDir)

	agent := agent.NewLogAgent(cfg, logger, flags.PluginDir)
	ctx, cancel := context.WithCancel(command.Context())
	service, err := newAgentService(agent, cancel)
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

func newDefaultLoggerAt(level zapcore.Level) *zap.SugaredLogger {
	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(level)
	logCfg.Sampling.Initial = 5
	logCfg.Sampling.Thereafter = 100
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.StacktraceKey = ""
	logCfg.EncoderConfig.TimeKey = "timestamp"
	logCfg.EncoderConfig.MessageKey = "message"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// logCfg := zap.NewDevelopmentConfig()
	// logCfg.EncoderConfig.TimeKey = ""
	// logCfg.EncoderConfig.CallerKey = ""
	// logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	baseLogger, _ := logCfg.Build()
	return baseLogger.Sugar()
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
			logger.Info(srv.ListenAndServe())
		}()

		wg.Add(1)
		go func() {
			<-ctx.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := srv.Shutdown(ctx)
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
			}
			defer f.Close()

			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal("could not start CPU profile: ", err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(flags.CPUProfileDuration):
			}
			pprof.StopCPUProfile()
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
			defer f.Close() // error handling omitted for example

			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		}()
	}

	return wg
}
