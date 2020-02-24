package plugin

import (
	"sync"

	"go.uber.org/zap"
)

func init() {
	RegisterConfig("logger", &LoggerConfig{})
}

type LoggerConfig struct {
	DefaultDestinationConfig `mapstructure:",squash"`
	Field                    string
}

func (c LoggerConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	plugin := &LoggerPlugin{
		DefaultDestination: c.DefaultDestinationConfig.Build(),
		config:             c,
		SugaredLogger:      logger.With("plugin_type", "logger", "plugin_id", c.DefaultDestinationConfig.ID),
	}

	return plugin, nil
}

type LoggerPlugin struct {
	DefaultDestination
	config LoggerConfig
	*zap.SugaredLogger
}

func (p *LoggerPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		for {
			entry, ok := <-p.DefaultDestination.Input()
			if !ok {
				// TODO flush logger
				return
			}

			p.Infow("Received log", "entry", entry)
		}
	}()

	return nil
}
