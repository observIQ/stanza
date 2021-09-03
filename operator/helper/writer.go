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
func (c WriterConfig) Build(bc operator.BuildContext) (WriterOperator, error) {
	basicOperator, err := c.BasicConfig.Build(bc)
	if err != nil {
		return WriterOperator{}, err
	}

	// Namespace all the output IDs
	namespacedIDs := c.OutputIDs.WithNamespace(bc)
	if len(namespacedIDs) == 0 {
		namespacedIDs = bc.DefaultOutputIDs
	}

	writer := WriterOperator{
		OutputIDs:     namespacedIDs,
		BasicOperator: basicOperator,
	}
	return writer, nil
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
			if err := operator.Process(ctx, e); err != nil {
				w.Errorf("error while writing entry: %s", err)
			}
			return
		}
		if err := operator.Process(ctx, e.Copy()); err != nil {
			w.Errorf("error while writing entry: %s", err)
		}
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

func (o OutputIDs) WithNamespace(bc operator.BuildContext) OutputIDs {
	namespacedIDs := make([]string, 0, len(o))
	for _, id := range o {
		namespacedIDs = append(namespacedIDs, bc.PrependNamespace(id))
	}
	return namespacedIDs
}

// UnmarshalJSON will unmarshal a string or array of strings to OutputIDs.
func (o *OutputIDs) UnmarshalJSON(bytes []byte) error {
	var value interface{}
	err := json.Unmarshal(bytes, &value)
	if err != nil {
		return err
	}

	ids, err := NewOutputIDsFromInterface(value)
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

	ids, err := NewOutputIDsFromInterface(value)
	if err != nil {
		return err
	}

	*o = ids
	return nil
}

// NewOutputIDsFromInterface creates a new OutputIDs object from an interface
func NewOutputIDsFromInterface(value interface{}) (OutputIDs, error) {
	if str, ok := value.(string); ok {
		return OutputIDs{str}, nil
	}

	if array, ok := value.([]interface{}); ok {
		return NewOutputIDsFromArray(array)
	}

	return nil, fmt.Errorf("value is not of type string or string array")
}

// NewOutputIDsFromArray creates a new OutputIDs object from an array
func NewOutputIDsFromArray(array []interface{}) (OutputIDs, error) {
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
