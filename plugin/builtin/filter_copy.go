package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("copy_filter", &CopyFilterConfig{})
}

// CopyFilterConfig is the configuration of a copy filter.
type CopyFilterConfig struct {
	base.PluginConfig `mapstructure:",squash" yaml:",inline"`
	OutputIDs         []string `mapstructure:"outputs" yaml:"outputs"`
}

// Build will build a copy filter plugin.
func (c CopyFilterConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return nil, err
	}

	copyFilter := &CopyFilter{
		Plugin:    p,
		outputIDs: c.OutputIDs,
	}

	return copyFilter, nil
}

// CopyFilter is a plugin that sends a copy of an entry to multiple outputs.
type CopyFilter struct {
	base.Plugin
	outputIDs []string
	outputs   []plugin.Consumer
	*zap.SugaredLogger
}

// Consume will copy and send a log entry to the connected outputs.
func (f *CopyFilter) Consume(entry *entry.Entry) error {
	for _, output := range f.outputs {
		// TODO should we block if one output can't keep up?
		err := output.Consume(copyEntry(entry))
		if err != nil {
			// TODO what should err behavior look like for copy?
			return err
		}
	}

	return nil
}

// Consumers will return all connected plugins.
func (f *CopyFilter) Consumers() []plugin.Consumer {
	return f.outputs
}

// SetConsumers will set the outputs of the copy operation.
func (f *CopyFilter) SetConsumers(consumers []plugin.Consumer) error {
	f.outputs = make([]plugin.Consumer, len(f.outputIDs))

	for _, outputID := range f.outputIDs {
		for _, consumer := range consumers {
			if outputID == consumer.ID() {
				f.outputs = append(f.outputs, consumer)
				break
			}
		}
	}

	if len(f.outputs) != len(f.outputIDs) {
		return fmt.Errorf("missing output plugins")
	}
	return nil
}

// CopyEntry clones a log entry.
func copyEntry(e *entry.Entry) *entry.Entry {
	newEntry := entry.Entry{}
	newEntry.Timestamp = e.Timestamp
	newEntry.Record = copyMap(e.Record)

	return &newEntry
}
