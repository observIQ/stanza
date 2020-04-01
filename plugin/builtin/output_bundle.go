package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("bundle_output", &BundleOutputConfig{})
}

// BundleOutputConfig is the configuration of a bundle output plugin.
type BundleOutputConfig struct {
	base.OutputConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a bundle output plugin.
func (c BundleOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	bundleOutput := &BundleOutput{
		OutputPlugin: outputPlugin,
	}

	return bundleOutput, nil
}

// BundleOutput is a plugin that outputs entries to a bundle.
type BundleOutput struct {
	base.OutputPlugin
	bundle *Bundle
}

// Consume will pipe an entry out of the pipeline.
func (o *BundleOutput) Consume(e *entry.Entry) error {
	return o.PipeOut(e)
}

// PipeOut sends an entry to the parent bundle outside of the pipeline.
func (o *BundleOutput) PipeOut(e *entry.Entry) error {
	return o.bundle.Output.Consume(e)
}

// SetBundle will set the parent bundle.
func (o *BundleOutput) SetBundle(bundle *Bundle) {
	o.bundle = bundle
}
