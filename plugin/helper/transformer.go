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
	BasicConfig  `yaml:",inline"`
	WriterConfig `yaml:",inline"`
	OnError      string `json:"on_error" yaml:"on_error"`
}

// Build will build a transformer plugin.
func (c TransformerConfig) Build(context plugin.BuildContext) (TransformerPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return TransformerPlugin{}, errors.WithDetails(err, "plugin_id", c.PluginID)
	}

	writerPlugin, err := c.WriterConfig.Build(context)
	if err != nil {
		return TransformerPlugin{}, errors.WithDetails(err, "plugin_id", c.PluginID)
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

	transformerPlugin := TransformerPlugin{
		BasicPlugin:  basicPlugin,
		WriterPlugin: writerPlugin,
		OnError:      c.OnError,
	}

	return transformerPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *TransformerConfig) SetNamespace(namespace string, exclusions ...string) {
	c.BasicConfig.SetNamespace(namespace, exclusions...)
	c.WriterConfig.SetNamespace(namespace, exclusions...)
}

// TransformerPlugin provides a basic implementation of a transformer plugin.
type TransformerPlugin struct {
	BasicPlugin
	WriterPlugin
	OnError string
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
	t.Write(ctx, newEntry)
	return nil
}

// HandleEntryError will handle an entry error using the on_error strategy.
func (t *TransformerPlugin) HandleEntryError(ctx context.Context, entry *entry.Entry, err error) error {
	t.Errorw("Failed to process entry", zap.Any("error", err), zap.Any("action", t.OnError), zap.Any("entry", entry))
	if t.OnError == SendOnError {
		t.Write(ctx, entry)
		return nil
	}
	return err
}

// TransformFunction is function that transforms an entry.
type TransformFunction = func(*entry.Entry) (*entry.Entry, error)

// SendOnError specifies an on_error mode for sending entries after an error.
const SendOnError = "send"

// DropOnError specifies an on_error mode for dropping entries after an error.
const DropOnError = "drop"
