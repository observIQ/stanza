package remove

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("remove", func() operator.Builder { return NewRemoveOperatorConfig("") })
}

// NewRemoveOperatorConfig creates a new restructure operator config with default values
func NewRemoveOperatorConfig(operatorID string) *RemoveOperatorConfig {
	return &RemoveOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "remove"),
	}
}

// RemoveOperatorConfig is the configuration of a restructure operator
type RemoveOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`

	Field entry.Field `mapstructure:"field"  json:"field" yaml:"field"`
}

// Build will build a Remove operator from the supplied configuration
func (c RemoveOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}
	if c.Field == entry.NewNilField() {
		return nil, fmt.Errorf("remove: field is empty")
	}

	removeOperator := &RemoveOperator{
		TransformerOperator: transformerOperator,
		Field:               c.Field,
	}

	return []operator.Operator{removeOperator}, nil
}

// RemoveOperator is an operator that deletes a field
type RemoveOperator struct {
	helper.TransformerOperator
	Field entry.Field
}

// Process will process an entry with a restructure transformation.
func (p *RemoveOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will apply the restructure operations to an entry
func (p *RemoveOperator) Transform(entry *entry.Entry) error {
	_, exist := entry.Delete(p.Field)
	if !exist {
		return fmt.Errorf("remove: field does not exist")
	}
	return nil
}
