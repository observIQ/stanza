package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("bundle_output", &BundleOutputConfig{})
}

// BundleOutputConfig is the configuration of a bundle output plugin.
type BundleOutputConfig struct {
	helper.BasicIdentityConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a bundle output plugin.
func (c BundleOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicIdentity, err := c.BasicIdentityConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	bundleOutput := &BundleOutput{
		BasicIdentity: basicIdentity,
	}

	return bundleOutput, nil
}

// BundleOutput is a plugin that outputs entries to a bundle.
type BundleOutput struct {
	helper.BasicIdentity
	helper.BasicLifecycle
	helper.BasicOutput
	bundle *Bundle
}

// Process will pipe an entry out of the bundle pipeline.
func (o *BundleOutput) Process(e *entry.Entry) error {
	return o.PipeOut(e)
}

// PipeOut pipes an entry to the output of the parent bundle.
func (o *BundleOutput) PipeOut(e *entry.Entry) error {
	return o.bundle.Output.Process(e)
}

// SetBundle will set the parent bundle.
func (o *BundleOutput) SetBundle(bundle *Bundle) {
	o.bundle = bundle
}
