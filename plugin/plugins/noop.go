package plugins

import (
	"fmt"
	"sync"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("noop", &NoopConfig{})
}

type NoopConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig  `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c *NoopConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
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

	plugin := &NoopParser{
		DefaultPlugin:    defaultPlugin,
		DefaultInputter:  defaultInputter,
		DefaultOutputter: defaultOutputter,
	}

	return plugin, nil
}

type NoopParser struct {
	pg.DefaultPlugin
	pg.DefaultInputter
	pg.DefaultOutputter
}

func (p *NoopParser) Start(wg *sync.WaitGroup) error {
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
