package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("drop_output", &DropOutputConfig{})
}

// DropOutputConfig is the configuration of a drop output plugin.
type DropOutputConfig struct {
	base.OutputConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a drop output plugin.
func (c *DropOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	return &DropOutput{outputPlugin}, nil
}

// DropOutput is a plugin that consumes and ignores incoming entries.
type DropOutput struct {
	base.OutputPlugin
}

// Consume will drop the incoming entry.
func (p *DropOutput) Consume(entry *entry.Entry) error {
	return nil
}
