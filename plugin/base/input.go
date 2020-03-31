package base

import (
	"fmt"

	"github.com/bluemedora/bplogagent/plugin"
)

// InputConfig defines how to configure and build a basic input plugin.
type InputConfig struct {
	PluginConfig `mapstructure:",squash" yaml:",inline"`
	OutputID     string `mapstructure:"output" yaml:"output"`
}

// Build will build a basic input plugin.
func (c InputConfig) Build(context plugin.BuildContext) (InputPlugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return InputPlugin{}, err
	}

	input := InputPlugin{
		Plugin:   p,
		OutputID: c.OutputID,
	}

	return input, nil
}

// InputPlugin is a plugin that is a producer, but not a consumer.
type InputPlugin struct {
	Plugin
	OutputID string
	Output   plugin.Consumer
}

// Consumers will return an array containing the plugin's connected output.
func (s InputPlugin) Consumers() []plugin.Consumer {
	return []plugin.Consumer{s.Output}
}

// SetConsumers will find and set the consumer that matches the output id.
func (s InputPlugin) SetConsumers(consumers []plugin.Consumer) error {
	if s.OutputID == "" {
		return nil
	}

	for _, consumer := range consumers {
		if consumer.ID() == s.OutputID {
			s.Output = consumer
		}
	}

	if s.Output == nil {
		return fmt.Errorf("missing output %s", s.OutputID)
	}

	return nil
}
