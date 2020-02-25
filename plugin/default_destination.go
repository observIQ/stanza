package plugin

import (
	"github.com/bluemedora/bplogagent/entry"
)

type DefaultDestinationConfig struct {
	PluginID   PluginID `mapstructure:"id"`
	Type       string   `mapstructure:"type"`
	BufferSize uint     `mapstructure:"buffer_size"`
}

func (c DefaultDestinationConfig) Build() DefaultDestination {
	bufferSize := c.BufferSize
	if bufferSize == 0 {
		bufferSize = 100
	}
	return DefaultDestination{
		config: c,
		input:  make(chan entry.Entry, bufferSize),
	}
}

func (c DefaultDestinationConfig) ID() PluginID {
	return c.PluginID
}

type DefaultDestination struct {
	config DefaultDestinationConfig
	input  EntryChannel
}

func (s *DefaultDestination) Input() EntryChannel {
	return s.input
}

func (s *DefaultDestination) ID() PluginID {
	return s.config.ID()
}

func (s *DefaultDestination) Type() string {
	return s.config.Type
}
