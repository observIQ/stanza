package main

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// newLogger creates a logger from the supplied flags.
// If the flags do not specify a log file, the logger will default to stdout.
func newLogger(flags RootFlags) *zap.Logger {
	if flags.LogFile == "" {
		return newStdLogger(flags)
	}

	return newFileLogger(flags)
}

// newFileLogger creates a new logger that writes to a file
func newFileLogger(flags RootFlags) *zap.Logger {
	writer := &lumberjack.Logger{
		Filename:   flags.LogFile,
		MaxSize:    flags.MaxLogSize,
		MaxBackups: flags.MaxLogBackups,
		MaxAge:     flags.MaxLogAge,
	}

	zapLevel := getZapLevel(flags.LogLevel)
	core := newWriterCore(writer, zapLevel)
	return zap.New(core)
}

// newStdLogger creates a new logger that writes to stdout
func newStdLogger(flags RootFlags) *zap.Logger {
	zapLevel := getZapLevel(flags.LogLevel)
	core := newStdCore(zapLevel)
	return zap.New(core)
}

// newWriterCore returns a new core for logging to an io.Writer
func newWriterCore(writer io.Writer, level zapcore.Level) zapcore.Core {
	encoder := defaultEncoder()
	syncer := zapcore.AddSync(writer)
	return zapcore.NewCore(encoder, syncer, level)
}

// newStdCore creates a new core for logging to stdout.
func newStdCore(level zapcore.Level) zapcore.Core {
	encoder := defaultEncoder()
	syncer := zapcore.Lock(os.Stdout)
	return zapcore.NewCore(encoder, syncer, level)
}

// defaultEncoder returns the default encoder for logging
func defaultEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.MessageKey = "message"
	return zapcore.NewJSONEncoder(encoderConfig)
}

// getZapLevel gets the zap level for the supplied string
// If the string does not match a known value, this will default to InfoLevel
func getZapLevel(level string) zapcore.Level {
	switch string(level) {
	case "debug", "DEBUG":
		return zapcore.DebugLevel
	case "info", "INFO":
		return zapcore.InfoLevel
	case "warn", "WARN":
		return zapcore.WarnLevel
	case "error", "ERROR":
		return zapcore.ErrorLevel
	case "panic", "PANIC":
		return zapcore.PanicLevel
	case "fatal", "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
