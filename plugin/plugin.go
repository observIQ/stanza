package plugin

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

type Plugin interface {
	ID() string
	Start(*sync.WaitGroup) error
}

type Source interface {
	Plugin
	Outputter
	Stop()
}

type Processor interface {
	Plugin
	Outputter
	Inputter
}

type Destination interface {
	Plugin
	Inputter
}

type Outputter interface {
	SetOutputs(map[string]chan<- entry.Entry) error
	Outputs() []chan<- entry.Entry
}

type Inputter interface {
	Input() chan entry.Entry
}

func BuildPlugins(configs []PluginConfig, logger *zap.SugaredLogger) ([]Plugin, error) {
	plugins := make([]Plugin, 0, len(configs))
	for _, config := range configs {
		plugin, err := config.Build(logger)
		if err != nil {
			return nil, fmt.Errorf("failed to build plugin with ID '%s': %s", config.ID(), err)
		}

		plugins = append(plugins, plugin)
	}

	err := setPluginOutputs(plugins)
	if err != nil {
		return nil, err
	}

	return plugins, nil
}

func setPluginOutputs(plugins []Plugin) error {
	processorInputs := make(map[string]chan<- entry.Entry)

	// Generate the list of input channels
	for _, plugin := range plugins {
		if inputter, ok := plugin.(Inputter); ok {
			// TODO check for duplicate IDs
			processorInputs[plugin.ID()] = inputter.Input()
		}
	}

	// Set the output channels using the generated lists
	for _, plugin := range plugins {
		if outputter, ok := plugin.(Outputter); ok {
			err := outputter.SetOutputs(processorInputs)
			if err != nil {
				return fmt.Errorf("failed to set outputs for plugin with ID %v: %s", plugin.ID(), err)
			}
		}
	}

	return nil
}
