package plugin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
)

type DefaultProcessorConfig struct {
	PluginID   PluginID `mapstructure:"id"`
	Output     PluginID
	Type       string
	BufferSize uint `mapstructure:"buffer_size"`
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

func (c DefaultProcessorConfig) ID() PluginID {
	return c.PluginID
}

type DefaultProcessor struct {
	config         DefaultProcessorConfig
	output         EntryChannel
	outputPluginID PluginID
	input          EntryChannel
}

func (p *DefaultProcessor) SetOutputs(outputRegistry map[PluginID]EntryChannel) error {
	outputChan, ok := outputRegistry[p.config.Output]
	if !ok {
		return fmt.Errorf("no plugin with ID %v found", p.config.Output)
	}

	p.output = outputChan
	p.outputPluginID = p.config.Output
	return nil
}

func (s *DefaultProcessor) Outputs() map[PluginID]EntryChannel {
	return map[PluginID]EntryChannel{s.outputPluginID: s.output}
}

func (s *DefaultProcessor) Output() EntryChannel {
	return s.output
}

func (s *DefaultProcessor) Input() EntryChannel {
	return s.input
}

func (s *DefaultProcessor) ID() PluginID {
	return s.config.ID()
}

func (s *DefaultProcessor) Type() string {
	return s.config.Type
}
