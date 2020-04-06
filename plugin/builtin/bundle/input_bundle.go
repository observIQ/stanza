package bundle

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("bundle_input", &BundleInputConfig{})
}

// BundleInputConfig is the configuration of a bundle input plugin.
type BundleInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
}

// Build will build a bundle input plugin.
func (c BundleInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	bundleInput := &BundleInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
	}

	return bundleInput, nil
}

// BundleInput is a plugin that represents the receiving point of a bundle.
type BundleInput struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicInput
}

// PipeIn is used by bundles to submit entries to the beginning of their bundle pipeline.
func (p *BundleInput) PipeIn(entry *entry.Entry) error {
	return p.Output.Process(entry)
}
