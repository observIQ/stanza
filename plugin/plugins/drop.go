package plugins

import (
	"fmt"
	"sync"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("drop", &DropOutputConfig{})
}

type DropOutputConfig struct {
	pg.DefaultPluginConfig   `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c *DropOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	dest := &DropOutput{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
	}

	return dest, nil
}

type DropOutput struct {
	pg.DefaultPlugin
	pg.DefaultInputter
}

func (p *DropOutput) Start(wg *sync.WaitGroup) error {
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
