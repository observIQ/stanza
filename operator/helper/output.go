package helper

import (
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
)

// NewOutputConfig creates a new output config
func NewOutputConfig(operatorID, operatorType string) OutputConfig {
	return OutputConfig{
		BasicConfig: NewBasicConfig(operatorID, operatorType),
	}
}

// OutputConfig provides a basic implementation of an output operator config.
type OutputConfig struct {
	BasicConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build an output operator.
func (c OutputConfig) Build(context operator.BuildContext) (OutputOperator, error) {
	basicOperator, err := c.BasicConfig.Build(context)
	if err != nil {
		return OutputOperator{}, err
	}

	outputOperator := OutputOperator{
		BasicOperator: basicOperator,
	}

	return outputOperator, nil
}

// OutputOperator provides a basic implementation of an output operator.
type OutputOperator struct {
	BasicOperator
}

// CanProcess will always return true for an output operator.
func (o *OutputOperator) CanProcess() bool {
	return true
}

// CanOutput will always return false for an output operator.
func (o *OutputOperator) CanOutput() bool {
	return false
}

// Outputs will always return an empty array for an output operator.
func (o *OutputOperator) Outputs() []operator.Operator {
	return []operator.Operator{}
}

// SetOutputs will return an error if called.
func (o *OutputOperator) SetOutputs(operators []operator.Operator) error {
	return errors.NewError(
		"Operator can not output, but is attempting to set an output.",
		"This is an unexpected internal error. Please submit a bug/issue.",
	)
}
