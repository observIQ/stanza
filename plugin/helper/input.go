package helper

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// InputConfig provides a basic implementation of an input plugin config.
type InputConfig struct {
	BasicConfig `yaml:",inline"`

	WriteTo  entry.Field `json:"write_to" yaml:"write_to"`
	OutputID string      `json:"output" yaml:"output"`
}

// Build will build a base producer.
func (c InputConfig) Build(context plugin.BuildContext) (InputPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return InputPlugin{}, err
	}

	if c.OutputID == "" {
		return InputPlugin{}, errors.NewError(
			"Plugin config is missing the `output` field. This field determines where to send incoming logs.",
			"Ensure that a valid `output` field exists on the plugin config.",
		)
	}

	inputPlugin := InputPlugin{
		BasicPlugin: basicPlugin,
		WriteTo:     c.WriteTo,
		OutputID:    c.OutputID,
	}

	return inputPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *InputConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.PluginID, exclusions) {
		c.PluginID = AddNamespace(c.PluginID, namespace)
	}

	if CanNamespace(c.OutputID, exclusions) {
		c.OutputID = AddNamespace(c.OutputID, namespace)
	}
}

// InputPlugin provides a basic implementation of an input plugin.
type InputPlugin struct {
	BasicPlugin
	WriteTo  entry.Field
	OutputID string
	Output   plugin.Plugin
}

// Write will create an entry using the write_to field.
func (i *InputPlugin) Write(value interface{}) *entry.Entry {
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

// CanOutput will always return true for an input plugin.
func (i *InputPlugin) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (i *InputPlugin) Outputs() []plugin.Plugin {
	return []plugin.Plugin{i.Output}
}

// SetOutputs will set the output plugin.
func (i *InputPlugin) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, i.OutputID)
	if err != nil {
		return err
	}

	i.Output = output
	return nil
}

// FindOutput will find the matching output plugin in a collection.
func FindOutput(plugins []plugin.Plugin, outputID string) (plugin.Plugin, error) {
	for _, plugin := range plugins {
		if plugin.ID() == outputID {
			if !plugin.CanProcess() {
				return nil, errors.NewError(
					"Plugin could not use its designated output.",
					"Ensure that the output is a plugin that can process logs (such as a parser or destination).",
					"output_id", outputID,
				)
			}

			return plugin, nil
		}
	}

	return nil, errors.NewError(
		"Plugin could not find its output plugin.",
		"Ensure that the output plugin is spelled correctly and defined in the config.",
		"output_id", outputID,
	)
}
