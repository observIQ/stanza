package helper

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// BasicInputConfig provides a basic implementation of an input config.
type BasicInputConfig struct {
	OutputID string `mapstructure:"output" yaml:"output"`
}

// Build will build a base producer.
func (c BasicInputConfig) Build() (BasicInput, error) {
	if c.OutputID == "" {
		return BasicInput{}, fmt.Errorf("missing field 'output'")
	}

	basicInput := BasicInput{
		OutputID: c.OutputID,
	}

	return basicInput, nil
}

// BasicInput provides a basic implementation of an input plugin.
type BasicInput struct {
	OutputID string
	Output   plugin.Plugin
}

// CanProcess will always return false for an input plugin.
func (i *BasicInput) CanProcess() bool {
	return false
}

// Process will always return an error if called.
func (i *BasicInput) Process(entry *entry.Entry) error {
	return fmt.Errorf("%s can not process entries", i.OutputID)
}

// CanOutput will always return true for an input plugin.
func (i *BasicInput) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (i *BasicInput) Outputs() []plugin.Plugin {
	return []plugin.Plugin{i.Output}
}

// SetOutputs will set the output plugin.
func (i *BasicInput) SetOutputs(plugins []plugin.Plugin) error {
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
				return nil, fmt.Errorf("%s can not be an output", outputID)
			}

			return plugin, nil
		}
	}
	return nil, fmt.Errorf("unable to find output %s", outputID)
}
