package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("bundle_input", &BundleInputConfig{})
}

// BundleInputConfig is the configuration of a bundle input plugin.
type BundleInputConfig struct {
	base.InputConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a bundle input plugin.
func (c BundleInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	bundleInput := &BundleInput{
		InputPlugin: inputPlugin,
	}

	return bundleInput, nil
}

// BundleInput is a plugin that represents the receiving point of a bundle.
type BundleInput struct {
	base.InputPlugin
}

// Consume will start entry processing in a bundle.
func (p *BundleInput) Consume(entry *entry.Entry) error {
	return p.Output.Consume(entry)
}
