package helper

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"go.uber.org/zap"
)

func NewInputConfig(pluginID, pluginType string) InputConfig {
	return InputConfig{
		WriterConfig: NewWriterConfig(pluginID, pluginType),
		WriteTo:      entry.NewRecordField(),
	}
}

// InputConfig provides a basic implementation of an input plugin config.
type InputConfig struct {
	WriterConfig `yaml:",inline"`
	WriteTo      entry.Field `json:"write_to" yaml:"write_to"`
}

// Build will build a base producer.
func (c InputConfig) Build(context plugin.BuildContext) (InputOperator, error) {
	writerOperator, err := c.WriterConfig.Build(context)
	if err != nil {
		return InputOperator{}, errors.WithDetails(err, "plugin_id", c.ID())
	}

	inputOperator := InputOperator{
		WriterOperator: writerOperator,
		WriteTo:        c.WriteTo,
	}

	return inputOperator, nil
}

// InputOperator provides a basic implementation of an input plugin.
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

// CanProcess will always return false for an input plugin.
func (i *InputOperator) CanProcess() bool {
	return false
}

// Process will always return an error if called.
func (i *InputOperator) Process(ctx context.Context, entry *entry.Entry) error {
	i.Errorw("Operator received an entry, but can not process", zap.Any("entry", entry))
	return errors.NewError(
		"Operator can not process logs.",
		"Ensure that plugin is not configured to receive logs from other plugins",
	)
}
