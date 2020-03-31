package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("noop_filter", &NoopFilterConfig{})
}

// NoopFilterConfig is the configuration of a noop plugin.
type NoopFilterConfig struct {
	base.FilterConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a noop plugin.
func (c *NoopFilterConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	filterPlugin, err := c.FilterConfig.Build(context)
	if err != nil {
		return nil, err
	}

	return &NoopFilter{filterPlugin}, nil
}

// NoopFilter is a plugin that performs no operations on an entry.
type NoopFilter struct {
	base.FilterPlugin
}

// Consume will forward the entry to the next output without any alterations.
func (p *NoopFilter) Consume(entry *entry.Entry) error {
	return p.Output.Consume(entry)
}
