package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("noop", &NoopConfig{})
}

type NoopConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
}

func (c *NoopConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	plugin := &NoopParser{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
	}

	return plugin, nil
}

type NoopParser struct {
	pg.DefaultPlugin
	pg.DefaultOutputter
}

func (p *NoopParser) Input(entry *entry.Entry) error {
	return p.Output(entry)
}
