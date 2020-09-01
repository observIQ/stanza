package helper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
)

// NewWriterConfig creates a new writer config
func NewWriterConfig(operatorID, operatorType string) WriterConfig {
	return WriterConfig{
		BasicConfig: NewBasicConfig(operatorID, operatorType),
	}
}

// WriterConfig is the configuration of a writer operator.
type WriterConfig struct {
	BasicConfig `yaml:",inline"`
	OutputIDs   OutputIDs `json:"output" yaml:"output"`
}

// Build will build a writer operator from the config.
func (c WriterConfig) Build(context operator.BuildContext) (WriterOperator, error) {
	basicOperator, err := c.BasicConfig.Build(context)
	if err != nil {
		return WriterOperator{}, err
	}

	writer := WriterOperator{
		OutputIDs:     c.OutputIDs,
		BasicOperator: basicOperator,
	}
	return writer, nil
}

// SetNamespace will namespace the output ids of the writer.
func (c *WriterConfig) SetNamespace(namespace string, exclusions ...string) {
	c.BasicConfig.SetNamespace(namespace, exclusions...)
	for i, outputID := range c.OutputIDs {
		if CanNamespace(outputID, exclusions) {
			c.OutputIDs[i] = AddNamespace(outputID, namespace)
		}
	}
}

// WriterOperator is an operator that can write to other operators.
type WriterOperator struct {
	BasicOperator
	OutputIDs       OutputIDs
	OutputOperators []operator.Operator
}

// Write will write an entry to the outputs of the operator.
func (w *WriterOperator) Write(ctx context.Context, e *entry.Entry) {
	for i, operator := range w.OutputOperators {
		if i == len(w.OutputOperators)-1 {
			_ = operator.Process(ctx, e)
			return
		}
		operator.Process(ctx, e.Copy())
	}
}

// CanOutput always returns true for a writer operator.
func (w *WriterOperator) CanOutput() bool {
	return true
}

// Outputs returns the outputs of the writer operator.
func (w *WriterOperator) Outputs() []operator.Operator {
	return w.OutputOperators
}

// SetOutputs will set the outputs of the operator.
func (w *WriterOperator) SetOutputs(operators []operator.Operator) error {
	outputOperators := make([]operator.Operator, 0)

	for _, operatorID := range w.OutputIDs {
		operator, ok := w.findOperator(operators, operatorID)
		if !ok {
			return fmt.Errorf("operator '%s' does not exist", operatorID)
		}

		if !operator.CanProcess() {
			return fmt.Errorf("operator '%s' can not process entries", operatorID)
		}

		outputOperators = append(outputOperators, operator)
	}

	// No outputs have been set, so use the next configured operator
	if len(w.OutputIDs) == 0 {
		currentOperatorIndex := -1
		for i, operator := range operators {
			if operator.ID() == w.ID() {
				currentOperatorIndex = i
				break
			}
		}
		if currentOperatorIndex == -1 {
			return fmt.Errorf("unexpectedly could not find self in array of operators")
		}
		nextOperatorIndex := currentOperatorIndex + 1
		if nextOperatorIndex == len(operators) {
			return fmt.Errorf("cannot omit output for the last operator in the pipeline")
		}
		nextOperator := operators[nextOperatorIndex]
		if !nextOperator.CanProcess() {
			return fmt.Errorf("operator '%s' cannot process entries, but it was selected as a receiver because 'output' was omitted", nextOperator.ID())
		}
		outputOperators = append(outputOperators, nextOperator)
	}

	w.OutputOperators = outputOperators
	return nil
}

// FindOperator will find an operator matching the supplied id.
func (w *WriterOperator) findOperator(operators []operator.Operator, operatorID string) (operator.Operator, bool) {
	for _, operator := range operators {
		if operator.ID() == operatorID {
			return operator, true
		}
	}
	return nil, false
}

// OutputIDs is a collection of operator IDs used as outputs.
type OutputIDs []string

// UnmarshalJSON will unmarshal a string or array of strings to OutputIDs.
func (o *OutputIDs) UnmarshalJSON(bytes []byte) error {
	var value interface{}
	err := json.Unmarshal(bytes, &value)
	if err != nil {
		return err
	}

	ids, err := o.fromInterface(value)
	if err != nil {
		return err
	}

	*o = ids
	return nil
}

// UnmarshalYAML will unmarshal a string or array of strings to OutputIDs.
func (o *OutputIDs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value interface{}
	err := unmarshal(&value)
	if err != nil {
		return err
	}

	ids, err := o.fromInterface(value)
	if err != nil {
		return err
	}

	*o = ids
	return nil
}

// fromInterface will parse OutputIDs from a raw interface.
func (o *OutputIDs) fromInterface(value interface{}) (OutputIDs, error) {
	if str, ok := value.(string); ok {
		return OutputIDs{str}, nil
	}

	if array, ok := value.([]interface{}); ok {
		return o.fromArray(array)
	}

	return nil, fmt.Errorf("value is not of type string or string array")
}

// fromArray will parse OutputIDs from a raw array.
func (o *OutputIDs) fromArray(array []interface{}) (OutputIDs, error) {
	ids := OutputIDs{}
	for _, rawValue := range array {
		strValue, ok := rawValue.(string)
		if !ok {
			return nil, fmt.Errorf("value in array is not of type string")
		}
		ids = append(ids, strValue)
	}
	return ids, nil
}
