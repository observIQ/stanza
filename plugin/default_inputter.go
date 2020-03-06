package plugin

import (
	"github.com/bluemedora/bplogagent/entry"
)

type DefaultInputterConfig struct {
	BufferSize uint `mapstructure:"buffer_size"`
}

// TODO returning an error for consistency with other build methods,
// but it's not really necessary
func (c DefaultInputterConfig) Build() (DefaultInputter, error) {
	bufferSize := c.BufferSize
	if bufferSize == 0 {
		bufferSize = 10 // TODO benchmark and think about a sane default
	}

	return DefaultInputter{
		InputChannel: make(chan entry.Entry, bufferSize),
	}, nil
}

func (c DefaultInputterConfig) IsInputter() {}

type DefaultInputter struct {
	InputChannel EntryChannel
}

func (s *DefaultInputter) Input() EntryChannel {
	return s.InputChannel
}
