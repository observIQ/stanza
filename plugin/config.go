package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/pipeline"
	yaml "gopkg.in/yaml.v2"
)

// Enforce that Config implements operator.Builder
var _ operator.Builder = (*Config)(nil)

// Config is the config values for the plugin
type Config struct {
	helper.WriterConfig
	Plugin     *Plugin                `json:"-" yaml:"-"`
	Parameters map[string]interface{} `json:",squash" yaml:",squash"`
}

// Build implements operator.MultiBuilder
func (c *Config) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	if bc.PluginDepth > 10 {
		return nil, errors.NewError("reached max plugin depth", "ensure that there are no recursive dependencies in plugins")
	}

	params := c.getRenderParams(bc)
	pipelineConfigBytes, err := c.Plugin.Render(params)
	if err != nil {
		return nil, err
	}

	var pipelineConfig struct {
		Pipeline pipeline.Config
	}
	if err := yaml.Unmarshal(pipelineConfigBytes, &pipelineConfig); err != nil {
		return nil, err
	}

	nbc := bc.WithSubNamespace(c.ID()).WithIncrementedDepth()
	return pipelineConfig.Pipeline.BuildOperators(nbc)
}

func (c *Config) getRenderParams(bc operator.BuildContext) map[string]interface{} {
	// Copy the parameters to avoid mutating them
	params := map[string]interface{}{}
	for k, v := range c.Parameters {
		params[k] = v
	}

	// Add ID and output to params
	params["input"] = bc.PrependNamespace(c.ID())
	params["id"] = c.ID()
	params["output"] = c.yamlOutputs(bc)
	return params
}

func (c *Config) yamlOutputs(bc operator.BuildContext) string {
	outputIDs := c.OutputIDs
	if len(outputIDs) == 0 {
		outputIDs = bc.DefaultOutputIDs
	}
	namespacedOutputs := make([]string, 0, len(outputIDs))
	for _, outputID := range outputIDs {
		namespacedOutputs = append(namespacedOutputs, bc.PrependNamespace(outputID))
	}
	return fmt.Sprintf("[%s]", strings.Join(namespacedOutputs, ","))
}

// UnmarshalJSON unmarshals JSON
func (c *Config) UnmarshalJSON(raw []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	return c.unmarshalMap(m)
}

// UnmarshalYAML unmarshals YAML
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}

	return c.unmarshalMap(m)
}

func (c *Config) unmarshalMap(m map[string]interface{}) error {
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

// MarshalYAML marshals YAML
func (c Config) MarshalYAML() (interface{}, error) {
	return c.toMap(), nil
}

// MarshalJSON marshals JSON
func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.toMap())
}

func (c Config) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range c.Parameters {
		m[k] = v
	}
	m["id"] = c.ID()
	m["type"] = c.Type()
	m["output"] = c.OutputIDs
	return m
}
