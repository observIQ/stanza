package plugin

import (
	"fmt"
)

type DefaultOutputterConfig struct {
	Output PluginID
}

func (c DefaultOutputterConfig) Build() (DefaultOutputter, error) {
	if c.Output == "" {
		return DefaultOutputter{}, fmt.Errorf("required field 'output' is missing")
	}

	return DefaultOutputter{
		outputPluginID: c.Output,
	}, nil
}

type DefaultOutputter struct {
	output         EntryChannel
	outputPluginID PluginID
}

func (p *DefaultOutputter) SetOutputs(outputRegistry map[PluginID]EntryChannel) error {
	outputChan, ok := outputRegistry[p.outputPluginID]
	if !ok {
		return fmt.Errorf("no plugin with ID %v found", p.outputPluginID)
	}

	p.output = outputChan

	return nil
}

func (s *DefaultOutputter) Outputs() map[PluginID]EntryChannel {
	return map[PluginID]EntryChannel{s.outputPluginID: s.output}
}

func (s *DefaultOutputter) Output() EntryChannel {
	return s.output
}
