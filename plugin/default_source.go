package plugin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
)

type DefaultSourceConfig struct {
	PluginID string `json:"id" yaml:"id" mapstructure:"id"`
	Output   string
}

func (c DefaultSourceConfig) ID() string {
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
	config DefaultSourceConfig
	output chan<- entry.Entry
}

func (p *DefaultSource) SetOutputs(outputRegistry map[string]chan<- entry.Entry) error {
	outputChan, ok := outputRegistry[p.config.Output]
	if !ok {
		return fmt.Errorf("no plugin with ID %v found", p.config.Output)
	}

	p.output = outputChan
	return nil
}

func (s *DefaultSource) Outputs() []chan<- entry.Entry {
	return []chan<- entry.Entry{s.output}
}

func (s *DefaultSource) ID() string {
	return s.config.ID()
}
