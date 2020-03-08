package plugins

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("drop", &DropOutputConfig{})
}

type DropOutputConfig struct {
	pg.DefaultPluginConfig `mapstructure:",squash" yaml:",inline"`
}

func (c *DropOutputConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	dest := &DropOutput{
		DefaultPlugin: defaultPlugin,
	}

	return dest, nil
}

type DropOutput struct {
	pg.DefaultPlugin
}

func (p *DropOutput) Input(entry *entry.Entry) error {
	return nil
}
