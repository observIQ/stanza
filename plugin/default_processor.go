package plugin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
)

type DefaultProcessorConfig struct {
	PluginID   string `json:"id" yaml:"id" mapstructure:"id"`
	Output     string
	BufferSize uint
}

func (c DefaultProcessorConfig) Build() DefaultProcessor {
	bufferSize := c.BufferSize
	if bufferSize == 0 {
		bufferSize = 100
	}
	return DefaultProcessor{
		config: c,
		input:  make(chan entry.Entry, bufferSize),
	}
}

func (c DefaultProcessorConfig) ID() string {
	return c.PluginID
}

type DefaultProcessor struct {
	config DefaultProcessorConfig
	output chan<- entry.Entry
	input  chan entry.Entry
}

func (p *DefaultProcessor) SetOutputs(outputRegistry map[string]chan<- entry.Entry) error {
	outputChan, ok := outputRegistry[p.config.Output]
	if !ok {
		return fmt.Errorf("no plugin with ID %v found", p.config.Output)
	}

	p.output = outputChan
	return nil
}

func (s *DefaultProcessor) Outputs() []chan<- entry.Entry {
	return []chan<- entry.Entry{s.output}
}

func (s *DefaultProcessor) Input() chan entry.Entry {
	return s.input
}

func (s *DefaultProcessor) Output() chan<- entry.Entry {
	return s.output
}

func (s *DefaultProcessor) ID() string {
	return s.config.ID()
}
