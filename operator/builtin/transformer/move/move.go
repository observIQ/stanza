package move

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("move", func() operator.Builder { return NewMoveOperatorConfig("") })
}

// NewMoveOperatorConfig creates a new move operator config with default values
func NewMoveOperatorConfig(operatorID string) *MoveOperatorConfig {
	return &MoveOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "move"),
	}
}

// MoveOperatorConfig is the configuration of a move operator
type MoveOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`
	From                     entry.Field `mapstructure:"from" yaml:"from"`
	To                       entry.Field `mapstructure:"to" yaml:"to"`
}

// Build will build a Move operator from the supplied configuration
func (c MoveOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.To == entry.NewNilField() || c.From == entry.NewNilField() {
		return nil, fmt.Errorf("move: missing to or from field")
	}

	moveOperator := &MoveOperator{
		TransformerOperator: transformerOperator,
		From:                c.From,
		To:                  c.To,
	}

	return []operator.Operator{moveOperator}, nil
}

// MoveOperator is an operator that moves a field's value to a new field
type MoveOperator struct {
	helper.TransformerOperator
	From entry.Field
	To   entry.Field
}

// Process will process an entry with a move transformation.
func (p *MoveOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will apply the move operation to an entry
func (p *MoveOperator) Transform(e *entry.Entry) error {
	val, exist := p.From.Delete(e)
	if !exist {
		return fmt.Errorf("move: field does not exist")
	}
	return p.To.Set(e, val)
}
