package helper

import (
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// BasicOutput provides a basic implementation of an output plugin.
type BasicOutput struct{}

// CanProcess will always return true for an output plugin.
func (o *BasicOutput) CanProcess() bool {
	return true
}

// CanOutput will always return false for an output plugin.
func (o *BasicOutput) CanOutput() bool {
	return false
}

// Outputs will always return an empty array for an output plugin.
func (o *BasicOutput) Outputs() []plugin.Plugin {
	return []plugin.Plugin{}
}

// SetOutputs will return an error if called.
func (o *BasicOutput) SetOutputs(plugins []plugin.Plugin) error {
	return errors.NewError(
		"Plugin can not output, but is attempting to set an output.",
		"This is an unexpected internal error. Please submit a bug/issue.",
	)
}
