package transformer

import (
	"context"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("filter", func() operator.Builder { return NewFilterOperatorConfig("") })
}

// NewFilterOperatorConfig creates a filter operator config with default values
func NewFilterOperatorConfig(operatorID string) *FilterOperatorConfig {
	return &FilterOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "filter"),
	}
}

// FilterOperatorConfig is the configuration of a filter operator
type FilterOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	Expression               string `json:"expr"   yaml:"expr"`
}

// Build will build a filter operator from the supplied configuration
func (c FilterOperatorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformer, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	compiledExpression, err := expr.Compile(c.Expression, expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", c.Expression, err)
	}

	filterOperator := &FilterOperator{
		TransformerOperator: transformer,
		expression:          compiledExpression,
	}

	return filterOperator, nil
}

// FilterOperator is an operator that filters entries based on matching expressions
type FilterOperator struct {
	helper.TransformerOperator
	expression *vm.Program
}

// Process will drop incoming entries that match the filter expression
func (f *FilterOperator) Process(ctx context.Context, entry *entry.Entry) error {
	env := helper.GetExprEnv(entry)
	defer helper.PutExprEnv(env)

	matches, err := vm.Run(f.expression, env)
	if err != nil {
		f.Errorf("Running expressing returned an error", zap.Error(err))
		return nil
	}

	filtered, ok := matches.(bool)
	if !ok {
		f.Errorf("Expression did not compile as a boolean")
		return nil
	}

	if !filtered {
		f.Write(ctx, entry)
	}

	return nil
}
