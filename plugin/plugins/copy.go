package plugins

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

func init() {
	pg.RegisterConfig("copy", &CopyConfig{})
}

type CopyConfig struct {
	pg.DefaultPluginConfig   `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig `mapstructure:",squash" yaml:",inline"`
	PluginOutputs            []pg.PluginID `mapstructure:"outputs"`
	Field                    string
}

func (c CopyConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	outputs := make([]pg.Inputter, 0)
	for _, outputID := range c.PluginOutputs {
		output, ok := context.Plugins[outputID]
		if !ok {
			return nil, fmt.Errorf("no output found with ID %s", outputID)
		}

		inputter, ok := output.(pg.Inputter)
		if !ok {
			return nil, fmt.Errorf("output with ID '%s' is not an inputter", outputID)
		}

		outputs = append(outputs, inputter)
	}

	plugin := &CopyPlugin{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		outputs:         outputs,
		SugaredLogger:   context.Logger.With("plugin_type", "copy", "plugin_id", c.PluginID),
	}

	return plugin, nil
}

func (c CopyConfig) Outputs() []pg.PluginID {
	return c.PluginOutputs
}

type CopyPlugin struct {
	pg.DefaultPlugin
	pg.DefaultInputter

	outputs []pg.Inputter
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

func (p *CopyPlugin) Outputs() []pg.Inputter {
	return p.outputs
}

func copyEntry(e entry.Entry) entry.Entry {
	newEntry := entry.Entry{}
	newEntry.Timestamp = e.Timestamp
	newEntry.Record = copyMap(e.Record)

	return newEntry
}
