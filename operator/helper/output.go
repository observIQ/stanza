package helper

import (
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
)

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

// SetNamespace will namespace the id and output of the operator config.
func (c *OutputConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.ID(), exclusions) {
		c.OperatorID = AddNamespace(c.ID(), namespace)
	}
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
