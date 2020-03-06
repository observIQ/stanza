package plugins

import (
	"fmt"
	"sync"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("bundle_input", &BundleInputConfig{})
}

type BundleInputConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c BundleInputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	if context.BundleInput == nil {
		return nil, fmt.Errorf("bundle_input plugin can only be used in the context of a bundle")
	}

	plugin := &BundleInput{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
		input:            context.BundleInput,
	}

	return plugin, nil
}

type BundleInput struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	input pg.EntryChannel
}

func (p *BundleInput) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()

		// TODO this is an unnecessary channel operation, but I couldn't
		// figure out how to construct the bundles without another plugin here.
		// It would be great if p.input and p.Output() were the same thing.
		// This same concept applies to `bundle_output`
		for {
			entry, ok := <-p.input
			if !ok {
				return
			}

			p.Output() <- entry
		}
	}()

	return nil
}
