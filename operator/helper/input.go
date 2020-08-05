package helper

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"go.uber.org/zap"
)

// NewInputConfig creates a new input config with default values.
func NewInputConfig(operatorID, operatorType string) InputConfig {
	return InputConfig{
		WriterConfig: NewWriterConfig(operatorID, operatorType),
		WriteTo:      entry.NewRecordField(),
	}
}

// InputConfig provides a basic implementation of an input operator config.
type InputConfig struct {
	WriterConfig `yaml:",inline"`
	WriteTo      entry.Field `json:"write_to" yaml:"write_to"`
}

// Build will build a base producer.
func (c InputConfig) Build(context operator.BuildContext) (InputOperator, error) {
	writerOperator, err := c.WriterConfig.Build(context)
	if err != nil {
		return InputOperator{}, errors.WithDetails(err, "operator_id", c.ID())
	}

	inputOperator := InputOperator{
		WriterOperator: writerOperator,
		WriteTo:        c.WriteTo,
	}

	return inputOperator, nil
}

// InputOperator provides a basic implementation of an input operator.
type InputOperator struct {
	WriterOperator
	WriteTo entry.Field
}

// NewEntry will create a new entry using the write_to field.
func (i *InputOperator) NewEntry(value interface{}) *entry.Entry {
	entry := entry.New()
	entry.Set(i.WriteTo, value)
	return entry
}

// CanProcess will always return false for an input operator.
func (i *InputOperator) CanProcess() bool {
	return false
}

// Process will always return an error if called.
func (i *InputOperator) Process(ctx context.Context, entry *entry.Entry) error {
	i.Errorw("Operator received an entry, but can not process", zap.Any("entry", entry))
	return errors.NewError(
		"Operator can not process logs.",
		"Ensure that operator is not configured to receive logs from other operators",
	)
}
