package helper

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"go.uber.org/zap"
)

// InputConfig provides a basic implementation of an input plugin config.
type InputConfig struct {
	BasicConfig  `yaml:",inline"`
	WriterConfig `yaml:",inline"`
	WriteTo      entry.Field `json:"write_to" yaml:"write_to"`
}

// Build will build a base producer.
func (c InputConfig) Build(context plugin.BuildContext) (InputPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return InputPlugin{}, errors.WithDetails(err, "plugin_id", c.PluginID)
	}

	writerPlugin, err := c.WriterConfig.Build(context)
	if err != nil {
		return InputPlugin{}, errors.WithDetails(err, "plugin_id", c.PluginID)
	}

	inputPlugin := InputPlugin{
		BasicPlugin:  basicPlugin,
		WriterPlugin: writerPlugin,
		WriteTo:      c.WriteTo,
	}

	return inputPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *InputConfig) SetNamespace(namespace string, exclusions ...string) {
	c.BasicConfig.SetNamespace(namespace, exclusions...)
	c.WriterConfig.SetNamespace(namespace, exclusions...)
}

// InputPlugin provides a basic implementation of an input plugin.
type InputPlugin struct {
	BasicPlugin
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
