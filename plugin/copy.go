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
	Outputs               []PluginID
	Field                 string
}

func (c CopyConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	plugin := &CopyPlugin{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		config:          c,
		SugaredLogger:   logger.With("plugin_type", "copy", "plugin_id", c.PluginID),
	}

	return plugin, nil
}

type CopyPlugin struct {
	DefaultPlugin
	DefaultInputter

	outputs map[PluginID]EntryChannel
	config  CopyConfig
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
				output <- copyEntry(entry)
			}
		}
	}()

	return nil
}

func (p *CopyPlugin) SetOutputs(inputRegistry map[PluginID]EntryChannel) error {
	outputs := make(map[PluginID]EntryChannel, len(p.config.Outputs))
	for _, outputID := range p.config.Outputs {
		output, ok := inputRegistry[outputID]
		if !ok {
			return fmt.Errorf("no plugin with ID %v found", outputID)
		}

		outputs[outputID] = output
	}

	p.outputs = outputs
	return nil
}

func (p *CopyPlugin) Outputs() map[PluginID]EntryChannel {
	return p.outputs
}

func copyEntry(e entry.Entry) entry.Entry {
	newEntry := entry.Entry{}
	newEntry.Timestamp = e.Timestamp
	newEntry.Record = copyMap(e.Record)

	return newEntry
}
