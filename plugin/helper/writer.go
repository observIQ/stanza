package helper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// WriterConfig is the configuration of a writer plugin.
type WriterConfig struct {
	OutputIDs OutputIDs `json:"output" yaml:"output"`
}

// Build will build a writer plugin from the config.
func (c WriterConfig) Build(context plugin.BuildContext) (WriterPlugin, error) {
	if len(c.OutputIDs) == 0 {
		return WriterPlugin{}, errors.NewError(
			"missing required `output` field",
			"ensure that the `output` field is defined",
		)
	}

	writer := WriterPlugin{
		OutputIDs: c.OutputIDs,
	}
	return writer, nil
}

// SetNamespace will namespace the output ids of the writer.
func (c *WriterConfig) SetNamespace(namespace string, exclusions ...string) {
	for i, outputID := range c.OutputIDs {
		if CanNamespace(outputID, exclusions) {
			c.OutputIDs[i] = AddNamespace(outputID, namespace)
		}
	}
}

// WriterPlugin is a plugin that can write to other plugins.
type WriterPlugin struct {
	OutputIDs     OutputIDs
	OutputPlugins []plugin.Plugin
}

// Write will write an entry to the outputs of the plugin.
func (w *WriterPlugin) Write(ctx context.Context, entry *entry.Entry) {
	for _, plugin := range w.OutputPlugins {
		_ = plugin.Process(ctx, entry)
	}
}

// CanOutput always returns true for a writer plugin.
func (w *WriterPlugin) CanOutput() bool {
	return true
}

// Outputs returns the outputs of the writer plugin.
func (w *WriterPlugin) Outputs() []plugin.Plugin {
	return w.OutputPlugins
}

// SetOutputs will set the outputs of the plugin.
func (w *WriterPlugin) SetOutputs(plugins []plugin.Plugin) error {
	outputPlugins := make([]plugin.Plugin, 0)

	for _, pluginID := range w.OutputIDs {
		plugin, ok := w.findPlugin(plugins, pluginID)
		if !ok {
			return fmt.Errorf("plugin `%s` does not exist", pluginID)
		}

		if !plugin.CanProcess() {
			return fmt.Errorf("plugin `%s` can not process entries", pluginID)
		}

		outputPlugins = append(outputPlugins, plugin)
	}

	w.OutputPlugins = outputPlugins
	return nil
}

// FindPlugin will find a plugin matching the supplied id.
func (w *WriterPlugin) findPlugin(plugins []plugin.Plugin, pluginID string) (plugin.Plugin, bool) {
	for _, plugin := range plugins {
		if plugin.ID() == pluginID {
			return plugin, true
		}
	}
	return nil, false
}

// OutputIDs is a collection of plugin IDs used as outputs.
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
