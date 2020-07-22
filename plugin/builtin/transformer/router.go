package transformer

import (
	"context"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("router", func() plugin.Builder { return NewRouterOperatorConfig("") })
}

func NewRouterOperatorConfig(pluginID string) *RouterOperatorConfig {
	return &RouterOperatorConfig{
		BasicConfig: helper.NewBasicConfig(pluginID, "router"),
	}
}

// RouterOperatorConfig is the configuration of a router plugin
type RouterOperatorConfig struct {
	helper.BasicConfig `yaml:",inline"`
	Routes             []*RouterOperatorRouteConfig `json:"routes" yaml:"routes"`
}

// RouterOperatorRouteConfig is the configuration of a route on a router plugin
type RouterOperatorRouteConfig struct {
	Expression string           `json:"expr"   yaml:"expr"`
	OutputIDs  helper.OutputIDs `json:"output" yaml:"output"`
}

// Build will build a router plugin from the supplied configuration
func (c RouterOperatorConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
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
		route := RouterOperatorRoute{
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

// SetNamespace will namespace the router plugin and the outputs contained in its routes
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

// RouterOperator is a plugin that routes entries based on matching expressions
type RouterOperator struct {
	helper.BasicOperator
	routes []*RouterOperatorRoute
}

// RouterOperatorRoute is a route on a router plugin
type RouterOperatorRoute struct {
	Expression      *vm.Program
	OutputIDs       helper.OutputIDs
	OutputOperators []plugin.Operator
}

// CanProcess will always return true for a router plugin
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
			for _, output := range route.OutputOperators {
				_ = output.Process(ctx, entry)
			}
			break
		}
	}

	return nil
}

// CanOutput will always return true for a router plugin
func (p *RouterOperator) CanOutput() bool {
	return true
}

// Outputs will return all connected plugins.
func (p *RouterOperator) Outputs() []plugin.Operator {
	outputs := make([]plugin.Operator, 0, len(p.routes))
	for _, route := range p.routes {
		outputs = append(outputs, route.OutputOperators...)
	}
	return outputs
}

// SetOutputs will set the outputs of the router plugin.
func (p *RouterOperator) SetOutputs(plugins []plugin.Operator) error {
	for _, route := range p.routes {
		outputOperators, err := p.findOperators(plugins, route.OutputIDs)
		if err != nil {
			return fmt.Errorf("failed to set outputs on route: %s", err)
		}
		route.OutputOperators = outputOperators
	}
	return nil
}

// findOperators will find a subset of plugins from a collection.
func (p *RouterOperator) findOperators(plugins []plugin.Operator, pluginIDs []string) ([]plugin.Operator, error) {
	result := make([]plugin.Operator, 0)
	for _, pluginID := range pluginIDs {
		plugin, err := p.findOperator(plugins, pluginID)
		if err != nil {
			return nil, err
		}
		result = append(result, plugin)
	}
	return result, nil
}

// findOperator will find a plugin from a collection.
func (p *RouterOperator) findOperator(plugins []plugin.Operator, pluginID string) (plugin.Operator, error) {
	for _, plugin := range plugins {
		if plugin.ID() == pluginID {
			return plugin, nil
		}
	}
	return nil, fmt.Errorf("plugin %s does not exist", pluginID)
}
