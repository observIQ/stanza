package plugin

import (
	"fmt"
	"sync"
)

func init() {
	RegisterConfig("noop", &NoopConfig{})
}

type NoopConfig struct {
	DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	DefaultInputterConfig  `mapstructure:",squash" yaml:",inline"`
	DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c *NoopConfig) Build(context BuildContext) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to build default outputter: %s", err)
	}

	plugin := &NoopOutput{
		DefaultPlugin:    defaultPlugin,
		DefaultInputter:  defaultInputter,
		DefaultOutputter: defaultOutputter,
	}

	return plugin, nil
}

type NoopOutput struct {
	DefaultPlugin
	DefaultInputter
	DefaultOutputter
}

func (p *NoopOutput) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			p.Output() <- entry
		}
	}()

	return nil
}
