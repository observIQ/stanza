package helper

import (
	"fmt"

	"github.com/bluemedora/bplogagent/plugin"
)

// BasicTransformerConfig provides a basic implementation of a transformer config.
type BasicTransformerConfig struct {
	OutputID string `mapstructure:"output" yaml:"output"`
}

// Build will build a base producer.
func (c BasicTransformerConfig) Build() (BasicTransformer, error) {
	if c.OutputID == "" {
		return BasicTransformer{}, fmt.Errorf("missing field 'output'")
	}

	basicTransformer := BasicTransformer{
		OutputID: c.OutputID,
	}

	return basicTransformer, nil
}

// BasicTransformer provides a basic implementation of a transformer plugin.
type BasicTransformer struct {
	OutputID string
	Output   plugin.Plugin
}

// CanProcess will always return true for a transformer plugin.
func (t *BasicTransformer) CanProcess() bool {
	return true
}

// CanOutput will always return true for an input plugin.
func (t *BasicTransformer) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (t *BasicTransformer) Outputs() []plugin.Plugin {
	return []plugin.Plugin{t.Output}
}

// SetOutputs will set the output plugin.
func (t *BasicTransformer) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, t.OutputID)
	if err != nil {
		return err
	}

	t.Output = output
	return nil
}
