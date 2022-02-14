package service

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	fileOutput string = "file"
	stdOutput  string = "stdout"
)

// LoggingConfig is the config for stanza's internal logging
type LoggingConfig struct {
	Output string             `yaml:"output"`
	Level  zapcore.Level      `yaml:"level"`
	File   *lumberjack.Logger `yaml:"file"`
}

// Validate checks that the logging config is valid.
func (l LoggingConfig) Validate() error {
	switch l.Output {
	case fileOutput:
		if l.File == nil {
			return fmt.Errorf("'file' key must be specified if file output is specified")
		}
		if l.File.Filename == "" {
			// We could allow this for the default lumberjack log filename in os.TempDir,
			// but it is likely a mistake to specify no filename
			return fmt.Errorf("file.filename must not be empty")
		}
	case stdOutput: // OK; No additional validation necessary
	default:
		return fmt.Errorf("unknown output type: %s", l.Output)
	}

	return nil
}

// DefaultLoggingConfig returns the default logging config
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Output: stdOutput,
		Level:  zap.InfoLevel,
		File: &lumberjack.Logger{
			Filename:   "stanza.log",
			MaxBackups: 5,
			MaxSize:    10,
			MaxAge:     7,
		},
	}
}

// newLogger creates a logger from the supplied flags.
// If the flags do not specify a log file, the logger will default to stdout.
func NewLogger(c LoggingConfig) *zap.Logger {
	if c.Output == stdOutput {
		return newStdLogger(c)
	}

	return newFileLogger(c)
}

// newFileLogger creates a new logger that writes to a file
func newFileLogger(c LoggingConfig) *zap.Logger {
	writer := c.File
	core := newWriterCore(writer, c.Level)
	return zap.New(core)
}

// newStdLogger creates a new logger that writes to stdout
func newStdLogger(c LoggingConfig) *zap.Logger {
	core := newStdCore(c.Level)
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
