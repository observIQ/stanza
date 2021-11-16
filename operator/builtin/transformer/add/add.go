package add

import (
	"context"
	"fmt"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("add", func() operator.Builder { return NewAddOperatorConfig("") })
}

// NewAddOperatorConfig creates a new add operator config with default values
func NewAddOperatorConfig(operatorID string) *AddOperatorConfig {
	return &AddOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "add"),
	}
}

// AddOperatorConfig is the configuration of an add operator
type AddOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`
	Field                    entry.Field `mapstructure:"field" json:"field" yaml:"field"`
	Value                    interface{} `mapstructure:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"`
}

// Build will build an add operator from the supplied configuration
func (c AddOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	addOperator := &AddOperator{
		TransformerOperator: transformerOperator,
		Field:               c.Field,
	}
	strVal, ok := c.Value.(string)
	if !ok || !isExpr(strVal) {
		addOperator.Value = c.Value
		return []operator.Operator{addOperator}, nil
	}
	exprStr := strings.TrimPrefix(strVal, "EXPR(")
	exprStr = strings.TrimSuffix(exprStr, ")")

	compiled, err := expr.Compile(exprStr, expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", c.IfExpr, err)
	}

	addOperator.program = compiled
	return []operator.Operator{addOperator}, nil
}

// AddOperator is an operator that adds a string value or an expression value
type AddOperator struct {
	helper.TransformerOperator

	Field   entry.Field
	Value   interface{}
	program *vm.Program
}

// Process will process an entry with a add transformation.
func (p *AddOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will apply the add operations to an entry
func (p *AddOperator) Transform(e *entry.Entry) error {
	if p.Value != nil {
		return e.Set(p.Field, p.Value)
	}
	if p.program != nil {
		env := helper.GetExprEnv(e)
		defer helper.PutExprEnv(env)

		result, err := vm.Run(p.program, env)
		if err != nil {
			return fmt.Errorf("evaluate value_expr: %s", err)
		}
		return e.Set(p.Field, result)
	}
	return fmt.Errorf("add: missing required field 'value'")
}

func isExpr(str string) bool {
	return strings.HasPrefix(str, "EXPR(") && strings.HasSuffix(str, ")")
}
