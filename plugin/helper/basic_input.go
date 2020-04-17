package helper

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// BasicInputConfig provides a basic implementation of an input config.
type BasicInputConfig struct {
	OutputID string `mapstructure:"output" yaml:"output"`
}

// Build will build a base producer.
func (c BasicInputConfig) Build() (BasicInput, error) {
	if c.OutputID == "" {
		return BasicInput{}, errors.NewError(
			"Plugin config is missing the `output` field.",
			"This error occurs when a plugin requires an output, but the `output` field is omitted in the config.",
			"Please add a valid output to the plugin config.",
		)
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
	return errors.NewError(
		"Plugin can not process logs.",
		"This error can occur when logs are accidentally sent to a plugin that is only meant to produce logs.",
		"Please ensure that plugin is not configured receive logs from other plugins",
	)
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
				return nil, errors.NewError(
					"Input plugin could not use its designated output.",
					"This error can occur when a user accidentally sets the output to a plugin that is only meant to produce logs.",
					"Please verify that the output is a plugin that can process logs (such as a parser or destination).",
					"output_id", outputID,
				)
			}

			return plugin, nil
		}
	}

	return nil, errors.NewError(
		"Input plugin could not find its output plugin.",
		"This error can occur when a user accidentally misspells the output id or has forgotten to include a plugin in the config.",
		"Please verify that the output is spelled correctly and defined in the config.",
		"output_id", outputID,
	)
}
