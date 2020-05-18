package commands

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"

	agent "github.com/bluemedora/bplogagent/agent"
	"github.com/bluemedora/bplogagent/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RootFlags struct {
	ConfigFiles []string
	PprofPort   int
	Debug       bool
}

func NewRootCmd() *cobra.Command {
	rootFlags := &RootFlags{}

	root := &cobra.Command{
		// TODO change these once we have some branding
		Use:   "bplogagent [-c ./config.yaml]",
		Short: "A log parser and router",
		Long:  "A log parser and router",
		Run:   func(command *cobra.Command, args []string) { runRoot(command, args, rootFlags) },
	}

	rootFlagSet := root.PersistentFlags()
	rootFlagSet.StringSliceVarP(&rootFlags.ConfigFiles, "config", "c", []string{"/etc/bplogagent/bplogagent.yaml"}, "path to a config file") // TODO default locations
	rootFlagSet.IntVar(&rootFlags.PprofPort, "pprof_port", 0, "listen port for pprof profiling")
	rootFlagSet.BoolVar(&rootFlags.Debug, "debug", false, "debug logging")

	graphFlags := &GraphFlags{
		RootFlags: rootFlags,
	}

	graph := &cobra.Command{
		Use:   "graph",
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

	agent := agent.NewLogAgent(cfg, logger)
	err = agent.Start()
	if err != nil {
		logger.Errorw("Failed to start log agent", zap.Any("error", err))
		os.Exit(1)
	}

	// Start the profiler http server
	if flags.PprofPort != 0 {
		runtime.SetBlockProfileRate(10000)
		go func() {
			logger.Info(http.ListenAndServe(fmt.Sprintf(":%d", flags.PprofPort), nil))
		}()
	}

	// Wait for interrupt or command cancelled
	ctx, cancel := context.WithCancel(command.Context())
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		logger.Debug("Received an interrupt signal. Attempting to shut down gracefully")
		cancel()
	}()
	<-ctx.Done()
	agent.Stop()
}

func newDefaultLoggerAt(level zapcore.Level) *zap.SugaredLogger {
	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(level)
	logCfg.Sampling.Initial = 5
	logCfg.Sampling.Thereafter = 100
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.StacktraceKey = ""
	// logCfg := zap.NewDevelopmentConfig()
	// logCfg.EncoderConfig.TimeKey = ""
	// logCfg.EncoderConfig.CallerKey = ""
	// logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	baseLogger, _ := logCfg.Build()
	return baseLogger.Sugar()
}
