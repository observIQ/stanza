package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("drop_out", &DropOutputConfig{})
}

// DropOutputConfig is the configuration of a drop output plugin.
type DropOutputConfig struct {
	helper.BasicIdentityConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a drop output plugin.
func (c DropOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicIdentity, err := c.BasicIdentityConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	dropOutput := &DropOutput{
		BasicIdentity: basicIdentity,
	}

	return dropOutput, nil
}

// DropOutput is a plugin that consumes and ignores incoming entries.
type DropOutput struct {
	helper.BasicIdentity
	helper.BasicLifecycle
	helper.BasicOutput
}

// Process will drop the incoming entry.
func (p *DropOutput) Process(entry *entry.Entry) error {
	return nil
}
