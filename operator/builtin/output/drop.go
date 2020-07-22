package output

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("drop_output", func() operator.Builder { return NewDropOutputConfig("") })
}

func NewDropOutputConfig(operatorID string) *DropOutputConfig {
	return &DropOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "drop_output"),
	}
}

// DropOutputConfig is the configuration of a drop output operator.
type DropOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
}

// Build will build a drop output operator.
func (c DropOutputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	dropOutput := &DropOutput{
		OutputOperator: outputOperator,
	}

	return dropOutput, nil
}

// DropOutput is a operator that consumes and ignores incoming entries.
type DropOutput struct {
	helper.OutputOperator
}

// Process will drop the incoming entry.
func (p *DropOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return nil
}
