package filter

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"go.uber.org/zap"
)

func init() {
	operator.Register("filter", func() operator.Builder { return NewFilterOperatorConfig("") })
}

var upperBound = big.NewInt(1000)

var randInt = rand.Int // allow override for testing

// NewFilterOperatorConfig creates a filter operator config with default values
func NewFilterOperatorConfig(operatorID string) *FilterOperatorConfig {
	return &FilterOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "filter"),
		DropRatio:         1,
	}
}

// FilterOperatorConfig is the configuration of a filter operator
type FilterOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	Expression               string  `json:"expr"   yaml:"expr"`
	DropRatio                float64 `json:"drop_ratio"   yaml:"drop_ratio"`
}

// Build will build a filter operator from the supplied configuration
func (c FilterOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformer, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	compiledExpression, err := expr.Compile(c.Expression, expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", c.Expression, err)
	}

	if c.DropRatio < 0.0 || c.DropRatio > 1.0 {
		return nil, fmt.Errorf("drop_ratio must be a number between 0 and 1")
	}

	filterOperator := &FilterOperator{
		TransformerOperator: transformer,
		expression:          compiledExpression,
		dropCutoff:          big.NewInt(int64(c.DropRatio * 1000)),
	}

	return []operator.Operator{filterOperator}, nil
}

// FilterOperator is an operator that filters entries based on matching expressions
type FilterOperator struct {
	helper.TransformerOperator
	expression *vm.Program
	dropCutoff *big.Int // [0..1000)
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
		return nil
	}

	i, err := randInt(rand.Reader, upperBound)
	if err != nil {
		return err
	}

	if i.Cmp(f.dropCutoff) >= 0 {
		f.Write(ctx, entry)
	}

	return nil
}
