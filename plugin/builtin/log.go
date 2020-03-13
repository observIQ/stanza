package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap/zapcore"
)

func init() {
	pg.RegisterConfig("log", &LogOutputConfig{})
}

type LogOutputConfig struct {
	pg.DefaultPluginConfig `mapstructure:",squash" yaml:",inline"`
	Level                  string `yaml:",omitempty"`
}

func (c LogOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	newLogger := context.Logger.With("plugin_type", "logger", "plugin_id", c.ID())

	if c.Level == "" {
		c.Level = "debug"
	}

	level := new(zapcore.Level)
	err := level.UnmarshalText([]byte(c.Level))
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

	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	plugin := &LogOutput{
		DefaultPlugin: defaultPlugin,
		logFunc:       logFunc,
	}

	return plugin, nil
}

type LogOutput struct {
	pg.DefaultPlugin

	logFunc func(string, ...interface{})
}

func (p *LogOutput) Input(entry *entry.Entry) error {
	p.logFunc("Received log", "entry", entry)
	return nil
}
