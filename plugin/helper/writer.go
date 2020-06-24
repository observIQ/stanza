package helper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// WriterConfig is the configuration of a writer plugin.
type WriterConfig struct {
	OutputIDs []string `json:"output" yaml:"output"`
}

// Build will build a writer plugin from the config.
func (c WriterConfig) Build(context plugin.BuildContext) (WriterPlugin, error) {
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

// UnmarshalJSON will unmarshal a writer config from JSON.
func (c *WriterConfig) UnmarshalJSON(bytes []byte) error {
	rawMap := map[string]interface{}{}
	err := json.Unmarshal(bytes, &rawMap)
	if err != nil {
		return err
	}

	outputInterface, ok := rawMap["output"]
	if !ok {
		return fmt.Errorf("missing required field `output`")
	}

	switch value := outputInterface.(type) {
	case string:
		c = &WriterConfig{[]string{value}}
	case []string:
		c = &WriterConfig{value}
	default:
		return fmt.Errorf("output is not of type string or array of strings")
	}

	return nil
}

// UnmarshalYAML will unmarshal a writer config from YAML.
func (c *WriterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawMap := map[string]interface{}{}
	err := unmarshal(&rawMap)
	if err != nil {
		return err
	}

	outputInterface, ok := rawMap["output"]
	if !ok {
		return fmt.Errorf("missing required field `output`")
	}

	switch value := outputInterface.(type) {
	case string:
		c = &WriterConfig{[]string{value}}
	case []string:
		c = &WriterConfig{value}
	default:
		return fmt.Errorf("output is not of type string or array of strings")
	}

	return nil
}

// WriterPlugin is a plugin that can write to other plugins.
type WriterPlugin struct {
	OutputIDs     []string
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
		plugin, ok := w.FindPlugin(plugins, pluginID)
		if !ok {
			return fmt.Errorf("output %s does not exist", pluginID)
		}

		if !plugin.CanProcess() {
			return fmt.Errorf("output %s can not process entries", pluginID)
		}

		outputPlugins = append(outputPlugins, plugin)
	}

	w.OutputPlugins = outputPlugins
	return nil
}

// FindPlugin will find a plugin matching the supplied id.
func (w *WriterPlugin) FindPlugin(plugins []plugin.Plugin, pluginID string) (plugin.Plugin, bool) {
	for _, plugin := range plugins {
		if plugin.ID() == pluginID {
			return plugin, true
		}
	}
	return nil, false
}
