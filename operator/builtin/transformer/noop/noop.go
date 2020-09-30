package noop

import (
	"context"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.RegisterOperator("noop", func() operator.Builder { return NewNoopOperatorConfig("") })
}

// NewNoopOperatorConfig creates a new noop operator config with default values
func NewNoopOperatorConfig(operatorID string) *NoopOperatorConfig {
	return &NoopOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "noop"),
	}
}

// NoopOperatorConfig is the configuration of a noop operator.
type NoopOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
}

// Build will build a noop operator.
func (c NoopOperatorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	noopOperator := &NoopOperator{
		TransformerOperator: transformerOperator,
	}

	return noopOperator, nil
}

// NoopOperator is an operator that performs no operations on an entry.
type NoopOperator struct {
	helper.TransformerOperator
}

// Process will forward the entry to the next output without any alterations.
func (p *NoopOperator) Process(ctx context.Context, entry *entry.Entry) error {
	p.Write(ctx, entry)
	return nil
}
