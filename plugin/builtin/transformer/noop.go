package transformer

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("noop", func() plugin.Builder { return NewNoopOperatorConfig("") })
}

func NewNoopOperatorConfig(pluginID string) *NoopOperatorConfig {
	return &NoopOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(pluginID, "noop"),
	}
}

// NoopOperatorConfig is the configuration of a noop plugin.
type NoopOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
}

// Build will build a noop plugin.
func (c NoopOperatorConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	noopOperator := &NoopOperator{
		TransformerOperator: transformerOperator,
	}

	return noopOperator, nil
}

// NoopOperator is a plugin that performs no operations on an entry.
type NoopOperator struct {
	helper.TransformerOperator
}

// Process will forward the entry to the next output without any alterations.
func (p *NoopOperator) Process(ctx context.Context, entry *entry.Entry) error {
	p.Write(ctx, entry)
	return nil
}
