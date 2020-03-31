package base

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// ParserConfig defines how to configure and build a basic parser plugin.
type ParserConfig struct {
	InputConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a basic parser plugin.
func (c ParserConfig) Build(context plugin.BuildContext) (ParserPlugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return ParserPlugin{}, err
	}

	return ParserPlugin{inputPlugin}, nil
}

// ParserPlugin is a plugin that parses a field in an entry.
type ParserPlugin struct {
	InputPlugin
}

// Consume will log that an entry has been parsed.
func (t *ParserPlugin) Consume(entry *entry.Entry) error {
	t.Debug("Entry parsed")
	return nil
}
