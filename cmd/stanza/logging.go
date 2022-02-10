package main

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

const (
	fileOutput string = "file"
	stdOutput  string = "stdout"
)

type loggingConfig struct {
	Output string             `yaml:"output"`
	Level  zapcore.Level      `yaml:"level"`
	File   *lumberjack.Logger `yaml:"file"`
}

// validate checks that the logging config is valid.
func (l loggingConfig) validate() error {
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

// defaultLoggingConfig returns the default logging config
func defaultLoggingConfig() loggingConfig {
	return loggingConfig{
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

// getLoggingConfig reads the config file specified by the flags into memory.
// Fields that aren't filled in the config are initialized to the defaults from
// defaultLoggingConfig.
// If no LogConfig is specified in the root flags, then the default config from defaultLoggingConfig is returned.
func getLoggingConfig(flags *RootFlags) (loggingConfig, error) {
	conf := defaultLoggingConfig()

	if flags.LogConfig != "" {
		f, err := os.Open(flags.LogConfig)
		if err != nil {
			return conf, err
		}

		buf, err := io.ReadAll(f)
		if err != nil {
			return conf, err
		}

		err = yaml.UnmarshalStrict(buf, &conf)
		if err != nil {
			return conf, err
		}
	}

	return conf, nil
}

// newLogger creates a logger from the supplied flags.
// If the flags do not specify a log file, the logger will default to stdout.
func newLogger(c loggingConfig) *zap.Logger {
	if c.Output == stdOutput {
		return newStdLogger(c)
	}

	return newFileLogger(c)
}

// newFileLogger creates a new logger that writes to a file
func newFileLogger(c loggingConfig) *zap.Logger {
	writer := c.File
	core := newWriterCore(writer, c.Level)
	return zap.New(core)
}

// newStdLogger creates a new logger that writes to stdout
func newStdLogger(c loggingConfig) *zap.Logger {
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
