package helper

import (
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
)

func NewOutputConfig(pluginID, pluginType string) OutputConfig {
	return OutputConfig{
		BasicConfig: NewBasicConfig(pluginID, pluginType),
	}
}

// OutputConfig provides a basic implementation of an output plugin config.
type OutputConfig struct {
	BasicConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build an output plugin.
func (c OutputConfig) Build(context plugin.BuildContext) (OutputOperator, error) {
	basicOperator, err := c.BasicConfig.Build(context)
	if err != nil {
		return OutputOperator{}, err
	}

	outputOperator := OutputOperator{
		BasicOperator: basicOperator,
	}

	return outputOperator, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *OutputConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.ID(), exclusions) {
		c.OperatorID = AddNamespace(c.ID(), namespace)
	}
}

// OutputOperator provides a basic implementation of an output plugin.
type OutputOperator struct {
	BasicOperator
}

// CanProcess will always return true for an output plugin.
func (o *OutputOperator) CanProcess() bool {
	return true
}

// CanOutput will always return false for an output plugin.
func (o *OutputOperator) CanOutput() bool {
	return false
}

// Outputs will always return an empty array for an output plugin.
func (o *OutputOperator) Outputs() []plugin.Operator {
	return []plugin.Operator{}
}

// SetOutputs will return an error if called.
func (o *OutputOperator) SetOutputs(plugins []plugin.Operator) error {
	return errors.NewError(
		"Operator can not output, but is attempting to set an output.",
		"This is an unexpected internal error. Please submit a bug/issue.",
	)
}
