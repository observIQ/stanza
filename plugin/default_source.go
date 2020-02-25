package plugin

import (
	"fmt"
)

type DefaultSourceConfig struct {
	PluginID PluginID `json:"id" yaml:"id" mapstructure:"id"`
	Output   PluginID
	Type     string
}

func (c DefaultSourceConfig) ID() PluginID {
	return c.PluginID
}

func (c DefaultSourceConfig) Build() (DefaultSource, error) {
	if c.PluginID == "" {
		return DefaultSource{}, fmt.Errorf("required field ID is missing")
	}

	if c.Output == "" {
		return DefaultSource{}, fmt.Errorf("required field ID is missing")
	}

	return DefaultSource{
		config: c,
	}, nil
}

type DefaultSource struct {
	config         DefaultSourceConfig
	output         EntryChannel
	outputPluginID PluginID
}

func (p *DefaultSource) SetOutputs(outputRegistry map[PluginID]EntryChannel) error {
	outputChan, ok := outputRegistry[p.config.Output]
	if !ok {
		return fmt.Errorf("no plugin with ID %v found", p.config.Output)
	}

	p.output = outputChan
	p.outputPluginID = p.config.Output

	return nil
}

func (s *DefaultSource) Outputs() map[PluginID]EntryChannel {
	return map[PluginID]EntryChannel{s.outputPluginID: s.output}
}

func (s *DefaultSource) ID() PluginID {
	return s.config.ID()
}

func (s *DefaultSource) Type() string {
	return s.config.Type
}
