package helper

import (
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"go.uber.org/zap"
)

func NewBasicConfig(pluginID, pluginType string) BasicConfig {
	return BasicConfig{
		OperatorID:   pluginID,
		OperatorType: pluginType,
	}
}

// BasicConfig provides a basic implemention for a plugin config.
type BasicConfig struct {
	OperatorID   string `json:"id"   yaml:"id"`
	OperatorType string `json:"type" yaml:"type"`
}

// ID will return the plugin id.
func (c BasicConfig) ID() string {
	if c.OperatorID == "" {
		return c.OperatorType
	}
	return c.OperatorID
}

// Type will return the plugin type.
func (c BasicConfig) Type() string {
	return c.OperatorType
}

// Build will build a basic plugin.
func (c BasicConfig) Build(context plugin.BuildContext) (BasicOperator, error) {
	if c.OperatorType == "" {
		return BasicOperator{}, errors.NewError(
			"missing required `type` field.",
			"ensure that all plugins have a uniquely defined `type` field.",
			"plugin_id", c.ID(),
		)
	}

	if context.Logger == nil {
		return BasicOperator{}, errors.NewError(
			"plugin build context is missing a logger.",
			"this is an unexpected internal error",
			"plugin_id", c.ID(),
			"plugin_type", c.Type(),
		)
	}

	plugin := BasicOperator{
		OperatorID:    c.ID(),
		OperatorType:  c.Type(),
		SugaredLogger: context.Logger.With("plugin_id", c.ID(), "plugin_type", c.Type()),
	}

	return plugin, nil
}

// SetNamespace will namespace the plugin id.
func (c *BasicConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.ID(), exclusions) {
		c.OperatorID = AddNamespace(c.ID(), namespace)
	}
}

// BasicOperator provides a basic implementation of a plugin.
type BasicOperator struct {
	OperatorID   string
	OperatorType string
	*zap.SugaredLogger
}

// ID will return the plugin id.
func (p *BasicOperator) ID() string {
	if p.OperatorID == "" {
		return p.OperatorType
	}
	return p.OperatorID
}

// Type will return the plugin type.
func (p *BasicOperator) Type() string {
	return p.OperatorType
}

// Logger returns the plugin's scoped logger.
func (p *BasicOperator) Logger() *zap.SugaredLogger {
	return p.SugaredLogger
}

// Start will start the plugin.
func (p *BasicOperator) Start() error {
	return nil
}

// Stop will stop the plugin.
func (p *BasicOperator) Stop() error {
	return nil
}
