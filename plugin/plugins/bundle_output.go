package plugins

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("bundle_output", &BundleOutputConfig{})
}

type BundleOutputConfig struct {
	pg.DefaultPluginConfig `mapstructure:",squash" yaml:",inline"`
	outputFunc             *func(*entry.Entry) error
}

func (c BundleOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	if !context.IsBundle {
		return nil, fmt.Errorf("bundle_output can only be used in context of a bundle")
	}

	plugin := &BundleOutput{
		DefaultPlugin: defaultPlugin,
	}

	return plugin, nil
}

type BundleOutput struct {
	pg.DefaultPlugin

	bundle BundleOutputter
}

func (p *BundleOutput) Input(entry *entry.Entry) error {
	return p.bundle.Output(entry)
}
