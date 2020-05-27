package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	agent "github.com/bluemedora/bplogagent/agent"
	"github.com/bluemedora/bplogagent/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RootFlags struct {
	ConfigFiles        []string
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
		// TODO change these once we have some branding
		Use:   "bplogagent [-c ./config.yaml]",
		Short: "A log parser and router",
		Long:  "A log parser and router",
		Args:  cobra.NoArgs,
		Run:   func(command *cobra.Command, args []string) { runRoot(command, args, rootFlags) },
	}

	rootFlagSet := root.PersistentFlags()
	rootFlagSet.StringSliceVarP(&rootFlags.ConfigFiles, "config", "c", []string{"/etc/bplogagent/bplogagent.yaml"}, "path to a config file") // TODO default locations
	rootFlagSet.BoolVar(&rootFlags.Debug, "debug", false, "debug logging")

	// Profiling flags
	rootFlagSet.IntVar(&rootFlags.PprofPort, "pprof_port", 0, "listen port for pprof profiling")
	rootFlagSet.MarkHidden("pprof_port")
	rootFlagSet.StringVar(&rootFlags.CPUProfile, "cpu_profile", "", "path to cpu profile output")
	rootFlagSet.MarkHidden("cpu_profile")
	rootFlagSet.DurationVar(&rootFlags.CPUProfileDuration, "cpu_profile_duration", 60*time.Second, "duration to run the cpu profile")
	rootFlagSet.MarkHidden("cpu_profile_duration")
	rootFlagSet.StringVar(&rootFlags.MemProfile, "mem_profile", "", "path to memory profile output")
	rootFlagSet.MarkHidden("mem_profile")
	rootFlagSet.DurationVar(&rootFlags.MemProfileDelay, "mem_profile_delay", 10*time.Second, "time to wait before writing a memory profile")
	rootFlagSet.MarkHidden("mem_profile_delay")

	graphFlags := &GraphFlags{
		RootFlags: rootFlags,
	}

	graph := &cobra.Command{
		Use:   "graph",
		Args:  cobra.NoArgs,
		Short: "Export a dot-formatted representation of the plugin graph",
		Run:   func(command *cobra.Command, args []string) { runGraph(command, args, graphFlags) },
	}

	root.AddCommand(graph)

	return root
}

func runRoot(command *cobra.Command, args []string, flags *RootFlags) {
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
		logger.Errorw("Failed to read config", zap.Any("error", err))
		os.Exit(1)
	}
	logger.Debugw("Parsed config", "config", cfg)

	agent := agent.NewLogAgent(cfg, logger)
	err = agent.Start()
	if err != nil {
		logger.Errorw("Failed to start log agent", zap.Any("error", err))
		os.Exit(1)
	}

	// Wait for interrupt or command cancelled
	ctx, cancel := context.WithCancel(command.Context())
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		logger.Info("Received an interrupt signal. Attempting to shut down gracefully")
		cancel()
	}()

	profilingWg := startProfiling(ctx, flags, logger)

	<-ctx.Done()
	agent.Stop()
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
