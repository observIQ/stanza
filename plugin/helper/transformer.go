package helper

import (
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// TransformerConfig provides a basic implementation of a transformer config.
type TransformerConfig struct {
	BasicConfig `yaml:",inline"`

	OutputID string `json:"output" yaml:"output"`
}

// ID will return the plugin id.
func (c TransformerConfig) ID() string {
	return c.PluginID
}

// Type will return the plugin type.
func (c TransformerConfig) Type() string {
	return c.PluginType
}

// Build will build a transformer plugin.
func (c TransformerConfig) Build(context plugin.BuildContext) (TransformerPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return TransformerPlugin{}, err
	}

	if c.OutputID == "" {
		return TransformerPlugin{}, errors.NewError(
			"Plugin config is missing the `output` field.",
			"Ensure that a valid `output` field exists on the plugin config.",
		)
	}

	transformerPlugin := TransformerPlugin{
		BasicPlugin: basicPlugin,
		OutputID:    c.OutputID,
	}

	return transformerPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *TransformerConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.PluginID, exclusions) {
		c.PluginID = AddNamespace(c.PluginID, namespace)
	}

	if CanNamespace(c.OutputID, exclusions) {
		c.OutputID = AddNamespace(c.OutputID, namespace)
	}
}

// TransformerPlugin provides a basic implementation of a transformer plugin.
type TransformerPlugin struct {
	BasicPlugin
	OutputID string
	Output   plugin.Plugin
}

// CanProcess will always return true for a transformer plugin.
func (t *TransformerPlugin) CanProcess() bool {
	return true
}

// CanOutput will always return true for an input plugin.
func (t *TransformerPlugin) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (t *TransformerPlugin) Outputs() []plugin.Plugin {
	return []plugin.Plugin{t.Output}
}

// SetOutputs will set the output plugin.
func (t *TransformerPlugin) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, t.OutputID)
	if err != nil {
		return err
	}

	t.Output = output
	return nil
}
