package base

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// FilterConfig defines how to configure and build a basic filter plugin.
type FilterConfig struct {
	InputConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a basic filter plugin.
func (c FilterConfig) Build(context plugin.BuildContext) (FilterPlugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return FilterPlugin{}, err
	}

	return FilterPlugin{inputPlugin}, nil
}

// FilterPlugin is a plugin that is plugin that filters log traffic.
type FilterPlugin struct {
	InputPlugin
}

// Consume will log that an entry has been filtered.
func (t *FilterPlugin) Consume(entry *entry.Entry) error {
	t.Debug("Entry filtered")
	return nil
}
