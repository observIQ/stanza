package builtin

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("copy", &CopyPluginConfig{})
}

// CopyPluginConfig is the configuration of a copy plugin.
type CopyPluginConfig struct {
	helper.BasicPluginConfig `yaml:",inline"`
	OutputIDs                []string `json:"outputs" yaml:"outputs"`
}

// Build will build a copy filter plugin.
func (c CopyPluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	copyPlugin := &CopyPlugin{
		BasicPlugin: basicPlugin,
		outputIDs:   c.OutputIDs,
	}

	return copyPlugin, nil
}

// CopyPlugin is a plugin that sends a copy of an entry to multiple outputs.
type CopyPlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	outputIDs []string
	outputs   []plugin.Plugin
	*zap.SugaredLogger
}

// CanProcess will always return true for a copy plugin.
func (p *CopyPlugin) CanProcess() bool {
	return true
}

// Process will copy and send a log entry to the connected outputs.
func (p *CopyPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	for _, output := range p.outputs {
		err := output.Process(ctx, copyEntry(entry))
		if err != nil {
			return err
		}
	}

	return nil
}

// CanOutput will always return true for a copy plugin.
func (p *CopyPlugin) CanOutput() bool {
	return true
}

// Outputs will return all connected plugins.
func (p *CopyPlugin) Outputs() []plugin.Plugin {
	return p.outputs
}

// SetOutputs will set the outputs of the copy plugin.
func (p *CopyPlugin) SetOutputs(plugins []plugin.Plugin) error {
	p.outputs = make([]plugin.Plugin, 0, len(p.outputIDs))

	for _, outputID := range p.outputIDs {
		output, err := helper.FindOutput(plugins, outputID)
		if err != nil {
			return err
		}
		p.outputs = append(p.outputs, output)
	}

	return nil
}

// CopyEntry clones a log entry.
func copyEntry(e *entry.Entry) *entry.Entry {
	newEntry := entry.Entry{}
	newEntry.Timestamp = e.Timestamp
	newEntry.Record = copyRecord(e.Record)

	return &newEntry
}
