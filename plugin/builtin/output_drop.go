package builtin

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("drop_output", &DropOutputConfig{})
}

// DropOutputConfig is the configuration of a drop output plugin.
type DropOutputConfig struct {
	helper.BasicPluginConfig `yaml:",inline"`
}

// Build will build a drop output plugin.
func (c DropOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	dropOutput := &DropOutput{
		BasicPlugin: basicPlugin,
	}

	return dropOutput, nil
}

// DropOutput is a plugin that consumes and ignores incoming entries.
type DropOutput struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicOutput
}

// Process will drop the incoming entry.
func (p *DropOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return nil
}
