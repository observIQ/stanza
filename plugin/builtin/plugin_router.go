package builtin

import (
	"context"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("router", &RouterPluginConfig{})
}

type RouterPluginConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	Routes                   []RouterPluginRouteConfig `mapstructure:"routes" json:"routes" yaml:"routes"`
}

type RouterPluginRouteConfig struct {
	Expression string `mapstructure:"expr" json:"expr" yaml:"expr"`
	Output     string `mapstructure:"output" json:"output" yaml:"output"`
}

func (c RouterPluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	routes := make([]*RouterPluginRoute, 0, len(c.Routes))
	for _, routeConfig := range c.Routes {
		compiled, err := expr.Compile(routeConfig.Expression, expr.AsBool(), expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression '%s': %w", routeConfig.Expression, err)
		}
		route := RouterPluginRoute{
			Expression: compiled,
			OutputID:   routeConfig.Output,
		}
		routes = append(routes, &route)
	}

	routerPlugin := &RouterPlugin{
		BasicPlugin: basicPlugin,
		routes:      routes,
	}

	return routerPlugin, nil
}

type RouterPlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle

	routes []*RouterPluginRoute
}

type RouterPluginRoute struct {
	Expression *vm.Program
	Output     plugin.Plugin
	OutputID   string
}

func (p *RouterPlugin) CanProcess() bool {
	return true
}

func (p *RouterPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	env := map[string]interface{}{
		"record": entry.Record,
	}

	for _, route := range p.routes {
		matches, err := vm.Run(route.Expression, env)
		if err != nil {
			p.Warnw("Running expression returned an error", zap.Error(err))
			continue
		}

		// we compile the expression with "AsBool", so this should be safe
		if matches.(bool) {
			err := route.Output.Process(ctx, entry)
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (p *RouterPlugin) CanOutput() bool {
	return true
}

// Outputs will return all connected plugins.
func (p *RouterPlugin) Outputs() []plugin.Plugin {
	outputs := make([]plugin.Plugin, 0, len(p.routes))
	for _, route := range p.routes {
		outputs = append(outputs, route.Output)
	}
	return outputs
}

// SetOutputs will set the outputs of the copy plugin.
func (p *RouterPlugin) SetOutputs(plugins []plugin.Plugin) error {
	for _, route := range p.routes {
		output, err := helper.FindOutput(plugins, route.OutputID)
		if err != nil {
			return err
		}
		route.Output = output
	}

	return nil
}
