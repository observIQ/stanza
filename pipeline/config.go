package pipeline

import (
	"fmt"
	"strings"

	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	yaml "gopkg.in/yaml.v2"
)

// Config is the configuration of a pipeline.
type Config []Params

// BuildPipeline will build a pipeline from the config.
func (c Config) BuildPipeline(context operator.BuildContext) (*Pipeline, error) {
	operatorConfigs, err := c.buildOperatorConfigs(context.PluginRegistry)
	if err != nil {
		return nil, err
	}

	operators, err := c.buildOperators(operatorConfigs, context)
	if err != nil {
		return nil, err
	}

	pipeline, err := NewPipeline(operators)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}

func (c Config) buildOperators(operatorConfigs []operator.Config, context operator.BuildContext) ([]operator.Operator, error) {
	operators := make([]operator.Operator, 0, len(operatorConfigs))
	for _, operatorConfig := range operatorConfigs {
		operator, err := operatorConfig.Build(context)

		if err != nil {
			return nil, errors.WithDetails(err,
				"operator_id", operatorConfig.ID(),
				"operator_type", operatorConfig.Type(),
			)
		}

		operators = append(operators, operator)
	}

	return operators, nil
}

func (c Config) buildOperatorConfigs(pluginRegistry operator.PluginRegistry) ([]operator.Config, error) {
	operatorConfigs := make([]operator.Config, 0, len(c))

	for _, params := range c {
		if err := params.Validate(); err != nil {
			return nil, errors.Wrap(err, "validate config params")
		}

		configs, err := params.BuildConfigs(pluginRegistry, "$")
		if err != nil {
			return nil, errors.Wrap(err, "build operator configs")
		}
		operatorConfigs = append(operatorConfigs, configs...)
	}

	return operatorConfigs, nil
}

// Params is a raw params map that can be converted into an operator config.
type Params map[string]interface{}

// ID returns the id field in the params map.
func (p Params) ID() string {
	if p.getString("id") == "" {
		return p.getString("type")
	}
	return p.getString("id")
}

// Type returns the type field in the params map.
func (p Params) Type() string {
	return p.getString("type")
}

// Outputs returns the output field in the params map.
func (p Params) Outputs() []string {
	return p.getStringArray("output")
}

// NamespacedID will return the id field with a namespace.
func (p Params) NamespacedID(namespace string) string {
	return helper.AddNamespace(p.ID(), namespace)
}

// NamespacedOutputs will return the output field with a namespace.
func (p Params) NamespacedOutputs(namespace string) []string {
	outputs := p.Outputs()
	for i, output := range outputs {
		outputs[i] = helper.AddNamespace(output, namespace)
	}
	return outputs
}

// TemplateInput will return the template input.
func (p Params) TemplateInput(namespace string) string {
	return helper.AddNamespace(p.ID(), namespace)
}

// TemplateOutput will return the template output.
func (p Params) TemplateOutput(namespace string) string {
	outputs := p.NamespacedOutputs(namespace)
	return fmt.Sprintf("[%s]", strings.Join(outputs[:], ", "))
}

// NamespaceExclusions will return all ids to exclude from namespacing.
func (p Params) NamespaceExclusions(namespace string) []string {
	exclusions := []string{p.NamespacedID(namespace)}
	for _, output := range p.NamespacedOutputs(namespace) {
		exclusions = append(exclusions, output)
	}
	return exclusions
}

// Validate will validate the basic fields required to make an operator config.
func (p Params) Validate() error {
	if p.Type() == "" {
		return errors.NewError(
			"missing required `type` field for operator config",
			"ensure that all operator configs have a defined type field",
			"id", p.ID(),
		)
	}

	return nil
}

// getString returns a string value from the params block.
func (p Params) getString(key string) string {
	rawValue, ok := p[key]
	if !ok {
		return ""
	}

	stringValue, ok := rawValue.(string)
	if !ok {
		return ""
	}

	return stringValue
}

// getStringArray returns a string array from the params block.
func (p Params) getStringArray(key string) []string {
	rawValue, ok := p[key]
	if !ok {
		return []string{}
	}

	switch value := rawValue.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []interface{}:
		result := []string{}
		for _, x := range value {
			if strValue, ok := x.(string); ok {
				result = append(result, strValue)
			}
		}
		return result
	default:
		return []string{}
	}
}

// BuildConfigs will build operator configs from a params map.
func (p Params) BuildConfigs(pluginRegistry operator.PluginRegistry, namespace string) ([]operator.Config, error) {
	if operator.IsDefined(p.Type()) {
		return p.buildAsBuiltin(namespace)
	}

	if pluginRegistry.IsDefined(p.Type()) {
		return p.buildPlugin(pluginRegistry, namespace)
	}

	return nil, errors.NewError(
		"unsupported `type` for operator config",
		"ensure that all operators have a supported builtin or plugin type",
		"type", p.Type(),
		"id", p.ID(),
	)
}

// buildAsBuiltin will build a builtin config from a params map.
func (p Params) buildAsBuiltin(namespace string) ([]operator.Config, error) {
	bytes, err := yaml.Marshal(p)
	if err != nil {
		return nil, errors.NewError(
			"failed to parse config map as yaml",
			"ensure that all config values are supported yaml values",
			"error", err.Error(),
		)
	}

	var config operator.Config
	if err := yaml.UnmarshalStrict(bytes, &config); err != nil {
		return nil, err
	}

	config.SetNamespace(namespace)
	return []operator.Config{config}, nil
}

// buildPlugin will build a plugin config from a params map.
func (p Params) buildPlugin(pluginRegistry operator.PluginRegistry, namespace string) ([]operator.Config, error) {
	templateParams := map[string]interface{}{}
	for key, value := range p {
		templateParams[key] = value
	}

	templateParams["input"] = p.TemplateInput(namespace)
	templateParams["output"] = p.TemplateOutput(namespace)

	config, err := pluginRegistry.Render(p.Type(), templateParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render plugin config")
	}

	exclusions := p.NamespaceExclusions(namespace)
	for _, operatorConfig := range config.Pipeline {
		innerNamespace := p.NamespacedID(namespace)
		operatorConfig.SetNamespace(innerNamespace, exclusions...)
	}

	return config.Pipeline, nil
}

// UnmarshalYAML will unmarshal yaml bytes into Params
func (p *Params) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[interface{}]interface{}
	err := unmarshal(&m)
	if err != nil {
		return err
	}

	*p = Params(cleanMap(m))
	return nil
}

func cleanMap(m map[interface{}]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(m))
	for k, v := range m {
		clean[fmt.Sprintf("%v", k)] = cleanValue(v)
	}
	return clean
}

func cleanValue(v interface{}) interface{} {
	switch v := v.(type) {
	case string, bool, int, int64, int32, float32, float64, nil:
		return v
	case map[interface{}]interface{}:
		return cleanMap(v)
	case []interface{}:
		res := make([]interface{}, 0, len(v))
		for _, arrayVal := range v {
			res = append(res, cleanValue(arrayVal))
		}
		return res
	default:
		return fmt.Sprintf("%v", v)
	}
}
