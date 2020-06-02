package builtin

import (
	"context"
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap/zapcore"
)

func init() {
	plugin.Register("logger_output", &LoggerOutputConfig{})
}

// LoggerOutputConfig is the configuration of a logger output plugin.
type LoggerOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`

	Level string `json:"level,omitempty" yaml:"level,omitempty"`
}

// Build will build a logger output plugin.
func (c LoggerOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	newLogger := context.Logger.With("plugin_type", "logger", "plugin_id", c.ID())
	if c.Level == "" {
		c.Level = "debug"
	}

	level := new(zapcore.Level)
	err = level.UnmarshalText([]byte(c.Level))
	if err != nil {
		return nil, fmt.Errorf("parse level: %s", err)
	}

	var logFunc func(string, ...interface{})
	switch *level {
	case zapcore.DebugLevel:
		logFunc = newLogger.Debugw
	case zapcore.InfoLevel:
		logFunc = newLogger.Infow
	case zapcore.WarnLevel:
		logFunc = newLogger.Warnw
	case zapcore.ErrorLevel:
		logFunc = newLogger.Errorw
	default:
		return nil, fmt.Errorf("log level '%s' is unsupported", level)
	}

	loggerOutput := &LoggerOutput{
		OutputPlugin: outputPlugin,
		logFunc:      logFunc,
	}

	return loggerOutput, nil
}

// LoggerOutput is a plugin that logs entries using the internal logger.
type LoggerOutput struct {
	helper.OutputPlugin
	logFunc func(string, ...interface{})
}

// Process will log entries received.
func (o *LoggerOutput) Process(ctx context.Context, entry *entry.Entry) error {
	o.logFunc("Received log", "entry", entry)
	return nil
}
