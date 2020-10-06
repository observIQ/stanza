//go:generate mockery --name=Builder --output=../testutil --outpkg=testutil --filename=operator_builder.go --structname=OperatorBuilder

package operator

import (
	"encoding/json"
	"fmt"
)

// Config is the configuration of an operator
type Config struct {
	MultiBuilder
}

// Identifier captures the methods needed to identify a builder or multibuilder
type Identifier interface {
	ID() string
	Type() string
}

// Builder is an entity that can build a single operator
type Builder interface {
	Identifier
	Build(BuildContext) (Operator, error)
}

// MultiBuilder is an entity that can build operators
type MultiBuilder interface {
	Identifier
	BuildMulti(BuildContext) ([]Operator, error)
}

// UnmarshalJSON will unmarshal a config from JSON.
func (c *Config) UnmarshalJSON(bytes []byte) error {
	var typeUnmarshaller struct {
		Type string
	}

	if err := json.Unmarshal(bytes, &typeUnmarshaller); err != nil {
		return err
	}

	if typeUnmarshaller.Type == "" {
		return fmt.Errorf("missing required field 'type'")
	}

	builderFunc, ok := DefaultRegistry.Lookup(typeUnmarshaller.Type)
	if !ok {
		return fmt.Errorf("unsupported type '%s'", typeUnmarshaller.Type)
	}

	builder := builderFunc()
	if err := json.Unmarshal(bytes, builder); err != nil {
		return fmt.Errorf("unmarshal to %s: %s", typeUnmarshaller.Type, err)
	}

	c.MultiBuilder = builder
	return nil
}

// MarshalJSON will marshal a config to JSON.
func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.MultiBuilder)
}

// UnmarshalYAML will unmarshal a config from YAML.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawConfig := map[string]interface{}{}
	err := unmarshal(&rawConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml to base config: %s", err)
	}

	typeInterface, ok := rawConfig["type"]
	if !ok {
		return fmt.Errorf("missing required field 'type'")
	}

	typeString, ok := typeInterface.(string)
	if !ok {
		return fmt.Errorf("non-string type %T for field 'type'", typeInterface)
	}

	builderFunc, ok := DefaultRegistry.Lookup(typeString)
	if !ok {
		return fmt.Errorf("unsupported type '%s'", typeString)
	}

	builder := builderFunc()
	if err = unmarshal(builder); err != nil {
		return fmt.Errorf("unmarshal to %s: %s", typeString, err)
	}

	c.MultiBuilder = builder
	return nil
}

// MarshalYAML will marshal a config to YAML.
func (c Config) MarshalYAML() (interface{}, error) {
	return c.MultiBuilder, nil
}
