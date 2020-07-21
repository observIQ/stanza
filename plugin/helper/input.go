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
func (c InputConfig) Build(context plugin.BuildContext) (InputPlugin, error) {
	writerPlugin, err := c.WriterConfig.Build(context)
	if err != nil {
		return InputPlugin{}, errors.WithDetails(err, "plugin_id", c.ID())
	}

	inputPlugin := InputPlugin{
		WriterPlugin: writerPlugin,
		WriteTo:      c.WriteTo,
	}

	return inputPlugin, nil
}

// InputPlugin provides a basic implementation of an input plugin.
type InputPlugin struct {
	WriterPlugin
	WriteTo entry.Field
}

// NewEntry will create a new entry using the write_to field.
func (i *InputPlugin) NewEntry(value interface{}) *entry.Entry {
	entry := entry.New()
	entry.Set(i.WriteTo, value)
	return entry
}

// CanProcess will always return false for an input plugin.
func (i *InputPlugin) CanProcess() bool {
	return false
}

// Process will always return an error if called.
func (i *InputPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	i.Errorw("Plugin received an entry, but can not process", zap.Any("entry", entry))
	return errors.NewError(
		"Plugin can not process logs.",
		"Ensure that plugin is not configured to receive logs from other plugins",
	)
}
