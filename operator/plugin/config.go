package plugin

import (
	"fmt"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/pipeline"
	yaml "gopkg.in/yaml.v2"
)

var _ operator.Builder = (*Config)(nil)

type Config struct {
	helper.BasicConfig
	plugin     *Plugin
	Parameters map[string]interface{} `json:",squash" yaml:",squash"`
}

func (c *Config) Build(bc operator.BuildContext) (operator.Operator, error) {
	params := c.getRenderParams()
	pipelineConfigBytes, err := c.plugin.Render(c.OperatorType, params)
	if err != nil {
		return nil, err
	}

	var pipelineConfig struct {
		Config pipeline.Config
	}
	if err := yaml.Unmarshal(pipelineConfigBytes, &pipelineConfig); err != nil {
		return nil, err
	}

	directedPipeline, err := pipelineConfig.Config.BuildPipeline(bc)
	if err != nil {
		return nil, err
	}

	basicOperator, err := c.BasicConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	var entrypoint operator.Operator
	for _, operator := range directedPipeline.Operators() {
		if operator.ID() == c.ID() {
			entrypoint = operator
			break
		}
	}

	return &PluginOperator{
		BasicOperator: basicOperator,
		Pipeline:      directedPipeline,
		Entrypoint:    entrypoint,
	}, nil
}

func (c *Config) getRenderParams() map[string]interface{} {
	// Copy the parameters to avoid mutating them
	params := map[string]interface{}{}
	for k, v := range c.Parameters {
		params[k] = v
	}

	// Add ID and output to params
	params["id"] = c.ID()
	params["output"] = c.yamlOutputs()
	return params
}

func (c *Config) yamlOutputs() string {
	// TODO
	return ""
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]interface{}
	if err := unmarshal(m); err != nil {
		return err
	}

	// TODO get outputs

	if id, ok := m["id"]; ok {
		if idString, ok := id.(string); ok {
			c.OperatorID = idString
		}
	}

	if t, ok := m["type"]; ok {
		if typeString, ok := t.(string); ok {
			c.OperatorType = typeString
		} else {
			return fmt.Errorf("invalid type %T for operator type", t)
		}
		return fmt.Errorf("missing required field 'type'")
	}

	return nil
}

func (c Config) MarshalYAML() {
	// TODO
}

func (c Config) UnmarshalJSON() {
	// TODO
}

func (c Config) MarshalJSON() {
	// TODO
}
