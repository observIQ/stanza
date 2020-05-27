package helper

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// BasicInputConfig provides a basic implementation of an input config.
type BasicInputConfig struct {
	WriteTo  entry.Field `json:"write_to" yaml:"write_to"`
	OutputID string      `json:"output" yaml:"output"`
}

// Build will build a base producer.
func (c BasicInputConfig) Build() (BasicInput, error) {
	if c.OutputID == "" {
		return BasicInput{}, errors.NewError(
			"Plugin config is missing the `output` field. This field determines where to send incoming logs.",
			"Ensure that a valid `output` field exists on the plugin config.",
		)
	}

	basicInput := BasicInput{
		WriteTo:  c.WriteTo,
		OutputID: c.OutputID,
	}

	return basicInput, nil
}

// BasicInput provides a basic implementation of an input plugin.
type BasicInput struct {
	WriteTo  entry.Field
	OutputID string
	Output   plugin.Plugin
}

// Write will create an entry using the write_to field and send it to the connected output.
func (i *BasicInput) Write(ctx context.Context, value interface{}) error {
	entry := entry.New()
	entry.Set(i.WriteTo, value)
	return i.Output.Process(ctx, entry)
}

// CanProcess will always return false for an input plugin.
func (i *BasicInput) CanProcess() bool {
	return false
}

// Process will always return an error if called.
func (i *BasicInput) Process(ctx context.Context, entry *entry.Entry) error {
	return errors.NewError(
		"Plugin can not process logs.",
		"Ensure that plugin is not configured to receive logs from other plugins",
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
					"Ensure that the output is a plugin that can process logs (such as a parser or destination).",
					"output_id", outputID,
				)
			}

			return plugin, nil
		}
	}

	return nil, errors.NewError(
		"Input plugin could not find its output plugin.",
		"Ensure that the output plugin is spelled correctly and defined in the config.",
		"output_id", outputID,
	)
}
