package router

import (
	"context"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("router", func() operator.Builder { return NewRouterOperatorConfig("") })
}

// NewRouterOperatorConfig config creates a new router operator config with default values
func NewRouterOperatorConfig(operatorID string) *RouterOperatorConfig {
	return &RouterOperatorConfig{
		BasicConfig: helper.NewBasicConfig(operatorID, "router"),
	}
}

// RouterOperatorConfig is the configuration of a router operator
type RouterOperatorConfig struct {
	helper.BasicConfig `yaml:",inline"`
	Routes             []*RouterOperatorRouteConfig `json:"routes" yaml:"routes"`
}

// RouterOperatorRouteConfig is the configuration of a route on a router operator
type RouterOperatorRouteConfig struct {
	helper.LabelerConfig `yaml:",inline"`
	Expression           string           `json:"expr"   yaml:"expr"`
	OutputIDs            helper.OutputIDs `json:"output" yaml:"output"`
}

// Build will build a router operator from the supplied configuration
func (c RouterOperatorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	basicOperator, err := c.BasicConfig.Build(context)
	if err != nil {
		return nil, err
	}

	routes := make([]*RouterOperatorRoute, 0, len(c.Routes))
	for _, routeConfig := range c.Routes {
		compiled, err := expr.Compile(routeConfig.Expression, expr.AsBool(), expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression '%s': %w", routeConfig.Expression, err)
		}

		labeler, err := routeConfig.LabelerConfig.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build labeler for route '%s': %w", routeConfig.Expression, err)
		}

		route := RouterOperatorRoute{
			Labeler:    labeler,
			Expression: compiled,
			OutputIDs:  routeConfig.OutputIDs,
		}
		routes = append(routes, &route)
	}

	routerOperator := &RouterOperator{
		BasicOperator: basicOperator,
		routes:        routes,
	}

	return routerOperator, nil
}

// SetNamespace will namespace the router operator and the outputs contained in its routes
func (c *RouterOperatorConfig) SetNamespace(namespace string, exclusions ...string) {
	c.BasicConfig.SetNamespace(namespace, exclusions...)
	for _, route := range c.Routes {
		for i, outputID := range route.OutputIDs {
			if helper.CanNamespace(outputID, exclusions) {
				route.OutputIDs[i] = helper.AddNamespace(outputID, namespace)
			}
		}
	}
}

// RouterOperator is an operator that routes entries based on matching expressions
type RouterOperator struct {
	helper.BasicOperator
	routes []*RouterOperatorRoute
}

// RouterOperatorRoute is a route on a router operator
type RouterOperatorRoute struct {
	helper.Labeler
	Expression      *vm.Program
	OutputIDs       helper.OutputIDs
	OutputOperators []operator.Operator
}

// CanProcess will always return true for a router operator
func (p *RouterOperator) CanProcess() bool {
	return true
}

// Process will route incoming entries based on matching expressions
func (p *RouterOperator) Process(ctx context.Context, entry *entry.Entry) error {
	env := helper.GetExprEnv(entry)
	defer helper.PutExprEnv(env)

	for _, route := range p.routes {
		matches, err := vm.Run(route.Expression, env)
		if err != nil {
			p.Warnw("Running expression returned an error", zap.Error(err))
			continue
		}

		// we compile the expression with "AsBool", so this should be safe
		if matches.(bool) {
			if err := route.Label(entry); err != nil {
				p.Errorf("Failed to label entry: %s", err)
				return err
			}

			for _, output := range route.OutputOperators {
				_ = output.Process(ctx, entry)
			}
			break
		}
	}

	return nil
}

// CanOutput will always return true for a router operator
func (p *RouterOperator) CanOutput() bool {
	return true
}

// Outputs will return all connected operators.
func (p *RouterOperator) Outputs() []operator.Operator {
	outputs := make([]operator.Operator, 0, len(p.routes))
	for _, route := range p.routes {
		outputs = append(outputs, route.OutputOperators...)
	}
	return outputs
}

// SetOutputs will set the outputs of the router operator.
func (p *RouterOperator) SetOutputs(operators []operator.Operator) error {
	for _, route := range p.routes {
		outputOperators, err := p.findOperators(operators, route.OutputIDs)
		if err != nil {
			return fmt.Errorf("failed to set outputs on route: %s", err)
		}
		route.OutputOperators = outputOperators
	}
	return nil
}

// findOperators will find a subset of operators from a collection.
func (p *RouterOperator) findOperators(operators []operator.Operator, operatorIDs []string) ([]operator.Operator, error) {
	result := make([]operator.Operator, 0)
	for _, operatorID := range operatorIDs {
		operator, err := p.findOperator(operators, operatorID)
		if err != nil {
			return nil, err
		}
		result = append(result, operator)
	}
	return result, nil
}

// findOperator will find an operator from a collection.
func (p *RouterOperator) findOperator(operators []operator.Operator, operatorID string) (operator.Operator, error) {
	for _, operator := range operators {
		if operator.ID() == operatorID {
			return operator, nil
		}
	}
	return nil, fmt.Errorf("operator %s does not exist", operatorID)
}
