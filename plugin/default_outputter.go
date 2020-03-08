package plugin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
)

// DefaultOutputterConfig
type DefaultOutputterConfig struct {
	Output PluginID
}

func (c DefaultOutputterConfig) Build(plugins map[PluginID]Plugin) (DefaultOutputter, error) {
	outputPlugin, ok := plugins[c.Output]
	if !ok {
		return DefaultOutputter{}, fmt.Errorf("could not find plugin with ID %s", c.Output)
	}

	inputter, ok := outputPlugin.(Inputter)
	if !ok {
		return DefaultOutputter{}, fmt.Errorf("plugin with ID '%s' is not an inputter, so can not be outputted to", outputPlugin.ID())
	}

	return DefaultOutputter{
		OutputPlugin: inputter,
	}, nil
}

func (c DefaultOutputterConfig) OutputIDs() []PluginID {
	return []PluginID{c.Output}
}

// DefaultOutputter
type DefaultOutputter struct {
	OutputPlugin Inputter
}

func (s *DefaultOutputter) Outputs() []Inputter {
	return []Inputter{s.OutputPlugin}
}

func (s *DefaultOutputter) Output(entry *entry.Entry) error {
	return s.OutputPlugin.Input(entry)
}
