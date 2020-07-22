package helper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
)

func NewWriterConfig(pluginID, pluginType string) WriterConfig {
	return WriterConfig{
		BasicConfig: NewBasicConfig(pluginID, pluginType),
	}
}

// WriterConfig is the configuration of a writer plugin.
type WriterConfig struct {
	BasicConfig `yaml:",inline"`
	OutputIDs   OutputIDs `json:"output" yaml:"output"`
}

// Build will build a writer plugin from the config.
func (c WriterConfig) Build(context plugin.BuildContext) (WriterOperator, error) {
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

// WriterOperator is a plugin that can write to other plugins.
type WriterOperator struct {
	BasicOperator
	OutputIDs       OutputIDs
	OutputOperators []plugin.Operator
}

// Write will write an entry to the outputs of the plugin.
func (w *WriterOperator) Write(ctx context.Context, e *entry.Entry) {
	for i, plugin := range w.OutputOperators {
		if i == len(w.OutputOperators)-1 {
			_ = plugin.Process(ctx, e)
			return
		}
		plugin.Process(ctx, e.Copy())
	}
}

// CanOutput always returns true for a writer plugin.
func (w *WriterOperator) CanOutput() bool {
	return true
}

// Outputs returns the outputs of the writer plugin.
func (w *WriterOperator) Outputs() []plugin.Operator {
	return w.OutputOperators
}

// SetOutputs will set the outputs of the plugin.
func (w *WriterOperator) SetOutputs(plugins []plugin.Operator) error {
	outputOperators := make([]plugin.Operator, 0)

	for _, pluginID := range w.OutputIDs {
		plugin, ok := w.findOperator(plugins, pluginID)
		if !ok {
			return fmt.Errorf("plugin '%s' does not exist", pluginID)
		}

		if !plugin.CanProcess() {
			return fmt.Errorf("plugin '%s' can not process entries", pluginID)
		}

		outputOperators = append(outputOperators, plugin)
	}

	// No outputs have been set, so use the next configured plugin
	if len(w.OutputIDs) == 0 {
		currentOperatorIndex := -1
		for i, plugin := range plugins {
			if plugin.ID() == w.ID() {
				currentOperatorIndex = i
				break
			}
		}
		if currentOperatorIndex == -1 {
			return fmt.Errorf("unexpectedly could not find self in array of plugins")
		}
		nextOperatorIndex := currentOperatorIndex + 1
		if nextOperatorIndex == len(plugins) {
			return fmt.Errorf("cannot omit output for the last plugin in the pipeline")
		}
		nextOperator := plugins[nextOperatorIndex]
		if !nextOperator.CanProcess() {
			return fmt.Errorf("plugin '%s' cannot process entries, but it was selected as a receiver because 'output' was omitted", nextOperator.ID())
		}
		outputOperators = append(outputOperators, nextOperator)
	}

	w.OutputOperators = outputOperators
	return nil
}

// FindOperator will find a plugin matching the supplied id.
func (w *WriterOperator) findOperator(plugins []plugin.Operator, pluginID string) (plugin.Operator, bool) {
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
