package plugins

import (
	"fmt"
	"sync"

	pg "github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap/zapcore"
)

func init() {
	pg.RegisterConfig("log", &LogOutputConfig{})
}

type LogOutputConfig struct {
	pg.DefaultPluginConfig   `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig `mapstructure:",squash" yaml:",inline"`
	Level                    string
}

func (c LogOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	newLogger := context.Logger.With("plugin_type", "logger", "plugin_id", c.ID())

	if c.Level == "" {
		c.Level = "debug"
	}

	level := new(zapcore.Level)
	err := level.UnmarshalText([]byte(c.Level))
	if err != nil {
		return nil, fmt.Errorf("failed to parse level: %s", err)
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
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	plugin := &LogOutput{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		logFunc:         logFunc,
	}

	return plugin, nil
}

type LogOutput struct {
	pg.DefaultPlugin
	pg.DefaultInputter

	logFunc func(string, ...interface{})
}

func (p *LogOutput) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		for {
			entry, ok := <-p.Input()
			if !ok {
				// TODO flush logger?
				return
			}

			p.logFunc("Received log", "entry", entry)
		}
	}()

	return nil
}
