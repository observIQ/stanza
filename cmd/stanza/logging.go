package main

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	registerWindowsSink()
}

func newDefaultLoggerAt(level zapcore.Level, path string) *zap.SugaredLogger {
	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(level)
	logCfg.Sampling = nil
	logCfg.EncoderConfig.CallerKey = ""
	logCfg.EncoderConfig.StacktraceKey = ""
	logCfg.EncoderConfig.TimeKey = "timestamp"
	logCfg.EncoderConfig.MessageKey = "message"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if path != "" {
		logCfg.OutputPaths = []string{pathToURI(path)}
	}

	baseLogger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	return baseLogger.Sugar()
}

func pathToURI(path string) string {
	switch runtime.GOOS {
	case "windows":
		return "winfile:///" + filepath.ToSlash(path)
	default:
		return filepath.ToSlash(path)
	}
}

var registerSyncsOnce sync.Once

func registerWindowsSink() {
	registerSyncsOnce.Do(func() {
		if runtime.GOOS == "windows" {
			err := zap.RegisterSink("winfile", newWinFileSink)
			if err != nil {
				panic(err)
			}
		}
	})
}

func newWinFileSink(u *url.URL) (zap.Sink, error) {
	// Ensure permissions restrict access to the running user only
	return os.OpenFile(u.Path[1:], os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
}
