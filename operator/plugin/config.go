package plugin

import (
	"fmt"
	"strings"

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
	OutputIDs  helper.OutputIDs
	// TODO outputs
}

func (c *Config) Build(bc operator.BuildContext) (operator.Operator, error) {
	nbc := bc.WithSubNamespace(c.ID())

	params := c.getRenderParams(bc)
	pipelineConfigBytes, err := c.plugin.Render(c.OperatorType, params)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Rendered %s:\n%s\n", c.ID(), pipelineConfigBytes)

	var pipelineConfig struct {
		Pipeline pipeline.Config
	}
	if err := yaml.Unmarshal(pipelineConfigBytes, &pipelineConfig); err != nil {
		return nil, err
	}

	directedPipeline, err := pipelineConfig.Pipeline.BuildPipeline(nbc)
	if err != nil {
		return nil, err
	}

	basicOperator, err := c.BasicConfig.Build(nbc)
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

func (c *Config) getRenderParams(bc operator.BuildContext) map[string]interface{} {
	// Copy the parameters to avoid mutating them
	params := map[string]interface{}{}
	for k, v := range c.Parameters {
		params[k] = v
	}

	// Add ID and output to params
	params["id"] = c.ID()
	params["output"] = c.yamlOutputs(bc)
	return params
}

func (c *Config) yamlOutputs(bc operator.BuildContext) string {
	namespacedOutputs := make([]string, 0, len(c.OutputIDs))
	for _, outputID := range c.OutputIDs {
		namespacedOutputs = append(namespacedOutputs, bc.PrependNamespace(outputID))
	}
	return fmt.Sprintf("[%s]", strings.Join(namespacedOutputs, ","))
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}

	if id, ok := m["id"]; ok {
		if idString, ok := id.(string); ok {
			c.OperatorID = idString
			delete(m, "id")
		}
	}

	if t, ok := m["type"]; ok {
		if typeString, ok := t.(string); ok {
			c.OperatorType = typeString
			delete(m, "type")
		} else {
			return fmt.Errorf("invalid type %T for operator type", t)
		}
	} else {
		return fmt.Errorf("missing required field 'type'")
	}

	if output, ok := m["output"]; ok {
		outputIDs, err := helper.NewOutputIDsFromInterface(output)
		if err != nil {
			return err
		}
		c.OutputIDs = outputIDs
		delete(m, "output")
	}

	c.Parameters = m
	return nil
}

func (c Config) MarshalYAML() (interface{}, error) {
	var m map[string]interface{}
	for k, v := range c.Parameters {
		m[k] = v
	}
	m["id"] = c.ID()
	m["type"] = c.Type()
	m["output"] = c.OutputIDs
	return m, nil
}

// func (c Config) UnmarshalJSON() {
// 	// TODO
// }

// func (c Config) MarshalJSON() {
// 	// TODO
// }
