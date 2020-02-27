package plugin

import "fmt"

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
		return DefaultOutputter{}, fmt.Errorf("plugin with ID '%s' is not an inputter, so can not be outputted to", inputter.ID())
	}

	return DefaultOutputter{
		outputPlugin: inputter,
	}, nil
}

func (c DefaultOutputterConfig) Outputs() []PluginID {
	return []PluginID{c.Output}
}

// DefaultOutputter
type DefaultOutputter struct {
	outputPlugin Inputter
}

func (s *DefaultOutputter) Outputs() []Inputter {
	return []Inputter{s.outputPlugin}
}

func (s *DefaultOutputter) Output() EntryChannel {
	return s.outputPlugin.Input()
}
