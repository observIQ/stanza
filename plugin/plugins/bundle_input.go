package plugins

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
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

	if !context.IsBundle {
		return nil, fmt.Errorf("bundle_output can only be used in context of a bundle")
	}

	plugin := &BundleInput{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
	}

	return plugin, nil
}

type BundleInput struct {
	pg.DefaultPlugin
	pg.DefaultOutputter
}

func (p *BundleInput) InputFromBundle(entry *entry.Entry) error {
	return p.Output(entry)
}
