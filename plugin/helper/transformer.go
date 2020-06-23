package helper

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// TransformerConfig provides a basic implementation of a transformer config.
type TransformerConfig struct {
	BasicConfig `yaml:",inline"`
	OnError     string `json:"on_error" yaml:"on_error"`
	OutputID    string `json:"output" yaml:"output"`
}

// Build will build a transformer plugin.
func (c TransformerConfig) Build(context plugin.BuildContext) (TransformerPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return TransformerPlugin{}, err
	}

	if c.OnError == "" {
		c.OnError = SendOnError
	}

	switch c.OnError {
	case SendOnError, DropOnError:
	default:
		return TransformerPlugin{}, errors.NewError(
			"plugin config has an invalid `on_error` field.",
			"ensure that the `on_error` field is set to either `send` or `drop`.",
			"on_error", c.OnError,
		)
	}

	if c.OutputID == "" {
		return TransformerPlugin{}, errors.NewError(
			"plugin config is missing the `output` field.",
			"ensure that a valid `output` field exists on the plugin config.",
		)
	}

	transformerPlugin := TransformerPlugin{
		BasicPlugin: basicPlugin,
		OnError:     c.OnError,
		OutputID:    c.OutputID,
	}

	return transformerPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *TransformerConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.PluginID, exclusions) {
		c.PluginID = AddNamespace(c.PluginID, namespace)
	}

	if CanNamespace(c.OutputID, exclusions) {
		c.OutputID = AddNamespace(c.OutputID, namespace)
	}
}

// TransformerPlugin provides a basic implementation of a transformer plugin.
type TransformerPlugin struct {
	BasicPlugin
	OnError  string
	OutputID string
	Output   plugin.Plugin
}

// CanProcess will always return true for a transformer plugin.
func (t *TransformerPlugin) CanProcess() bool {
	return true
}

// ProcessWith will process an entry with a transform function.
func (t *TransformerPlugin) ProcessWith(ctx context.Context, entry *entry.Entry, transform TransformFunction) error {
	newEntry, err := transform(entry)
	if err != nil {
		return t.HandleEntryError(ctx, entry, err)
	}
	return t.Output.Process(ctx, newEntry)
}

// HandleEntryError will handle an entry error using the on_error strategy.
func (t *TransformerPlugin) HandleEntryError(ctx context.Context, entry *entry.Entry, err error) error {
	t.Errorw("Failed to process entry", zap.Any("error", err), zap.Any("action", t.OnError), zap.Any("entry", entry))
	if t.OnError == SendOnError {
		return t.Output.Process(ctx, entry)
	}
	return err
}

// CanOutput will always return true for an input plugin.
func (t *TransformerPlugin) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (t *TransformerPlugin) Outputs() []plugin.Plugin {
	return []plugin.Plugin{t.Output}
}

// SetOutputs will set the output plugin.
func (t *TransformerPlugin) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, t.OutputID)
	if err != nil {
		return err
	}

	t.Output = output
	return nil
}

// TransformFunction is function that transforms an entry.
type TransformFunction = func(*entry.Entry) (*entry.Entry, error)

// SendOnError specifies an on_error mode for sending entries after an error.
const SendOnError = "send"

// DropOnError specifies an on_error mode for dropping entries after an error.
const DropOnError = "drop"
