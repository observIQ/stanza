package plugin

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

func init() {
	RegisterConfig("null", &NullOutputConfig{})
}

type NullOutputConfig struct {
	DefaultPluginConfig   `mapstructure:",squash"`
	DefaultInputterConfig `mapstructure:",squash"`
}

func (c *NullOutputConfig) Build(plugins map[PluginID]Plugin, logger *zap.SugaredLogger) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	dest := &NullOutput{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
	}

	return dest, nil
}

type NullOutput struct {
	DefaultPlugin
	DefaultInputter
}

func (p *NullOutput) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		for {
			_, ok := <-p.Input()
			if !ok {
				return
			}
		}
	}()

	return nil
}
