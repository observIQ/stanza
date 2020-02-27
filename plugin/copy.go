package plugin

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

func init() {
	RegisterConfig("copy", &CopyConfig{})
}

type CopyConfig struct {
	DefaultPluginConfig   `mapstructure:",squash"`
	DefaultInputterConfig `mapstructure:",squash"`
	PluginOutputs         []PluginID `mapstructure:"outputs"`
	Field                 string
}

func (c CopyConfig) Build(plugins map[PluginID]Plugin, logger *zap.SugaredLogger) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	outputs := make([]Inputter, 0)
	for _, outputID := range c.PluginOutputs {
		output, ok := plugins[outputID]
		if !ok {
			return nil, fmt.Errorf("no output found with ID %s", outputID)
		}

		inputter, ok := output.(Inputter)
		if !ok {
			return nil, fmt.Errorf("output with ID '%s' is not an inputter", outputID)
		}

		outputs = append(outputs, inputter)
	}

	plugin := &CopyPlugin{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		outputs:         outputs,
		SugaredLogger:   logger.With("plugin_type", "copy", "plugin_id", c.PluginID),
	}

	return plugin, nil
}

func (c CopyConfig) Outputs() []PluginID {
	return c.PluginOutputs
}

type CopyPlugin struct {
	DefaultPlugin
	DefaultInputter

	outputs []Inputter
	*zap.SugaredLogger
}

func (p *CopyPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			for _, output := range p.outputs {
				// TODO should we block if one output can't keep up?
				output.Input() <- copyEntry(entry)
			}
		}
	}()

	return nil
}

func (p *CopyPlugin) Outputs() []Inputter {
	return p.outputs
}

func copyEntry(e entry.Entry) entry.Entry {
	newEntry := entry.Entry{}
	newEntry.Timestamp = e.Timestamp
	newEntry.Record = copyMap(e.Record)

	return newEntry
}
