package helper

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"go.uber.org/zap"
)

func NewTransformerConfig(pluginID, pluginType string) TransformerConfig {
	return TransformerConfig{
		WriterConfig: NewWriterConfig(pluginID, pluginType),
		OnError:      SendOnError,
	}
}

// TransformerConfig provides a basic implementation of a transformer config.
type TransformerConfig struct {
	WriterConfig `yaml:",inline"`
	OnError      string `json:"on_error" yaml:"on_error"`
}

// Build will build a transformer plugin.
func (c TransformerConfig) Build(context plugin.BuildContext) (TransformerPlugin, error) {
	writerPlugin, err := c.WriterConfig.Build(context)
	if err != nil {
		return TransformerPlugin{}, errors.WithDetails(err, "plugin_id", c.ID())
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
		WriterPlugin: writerPlugin,
		OnError:      c.OnError,
	}

	return transformerPlugin, nil
}

// TransformerPlugin provides a basic implementation of a transformer plugin.
type TransformerPlugin struct {
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
