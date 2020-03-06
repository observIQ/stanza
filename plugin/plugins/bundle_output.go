package plugins

import (
	"fmt"
	"sync"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("bundle_output", &BundleOutputConfig{})
}

type BundleOutputConfig struct {
	pg.DefaultPluginConfig   `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c BundleOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	if context.BundleOutput == nil {
		return nil, fmt.Errorf("bundle_output plugin can only be used in the context of a bundle")
	}

	plugin := &BundleOutput{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		output:          context.BundleOutput,
	}

	return plugin, nil
}

type BundleOutput struct {
	pg.DefaultPlugin
	pg.DefaultInputter

	output pg.EntryChannel
}

func (p *BundleOutput) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			p.output <- entry
		}
	}()

	return nil
}
