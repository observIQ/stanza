//go:generate mockery --name=Builder --output=../testutil --outpkg=testutil --filename=operator_builder.go --structname=OperatorBuilder

package operator

import (
	"encoding/json"
	"fmt"

	"github.com/observiq/stanza/database"
	"go.uber.org/zap"
)

// Config is the configuration of an operator
type Config struct {
	Builder
}

// Builder is an entity that can build operators
type Builder interface {
	ID() string
	Type() string
	Build(BuildContext) (Operator, error)
	SetNamespace(namespace string, exclude ...string)
}

// BuildContext supplies contextual resources when building an operator.
type BuildContext struct {
	Database   database.Database
	Parameters map[string]interface{}
	Logger     *zap.SugaredLogger
}

// registry is a global registry of operator types to operator builders.
var registry = make(map[string]func() Builder)

// Register will register a function to an operator type.
// This function will return a builder for the supplied type.
func Register(operatorType string, newBuilder func() Builder) {
	registry[operatorType] = newBuilder
}

// IsDefined will return a boolean indicating if an operator type is registered and defined.
func IsDefined(operatorType string) bool {
	_, ok := registry[operatorType]
	return ok
}

// UnmarshalJSON will unmarshal a config from JSON.
func (c *Config) UnmarshalJSON(bytes []byte) error {
	var baseConfig struct {
		ID   string
		Type string
	}

	err := json.Unmarshal(bytes, &baseConfig)
	if err != nil {
		return err
	}

	if baseConfig.Type == "" {
		return fmt.Errorf("missing required field 'type'")
	}

	builderFunc, ok := registry[baseConfig.Type]
	if !ok {
		return fmt.Errorf("unsupported type '%s'", baseConfig.Type)
	}

	builder := builderFunc()
	err = json.Unmarshal(bytes, builder)
	if err != nil {
		return fmt.Errorf("unmarshal to %s: %s", baseConfig.Type, err)
	}

	c.Builder = builder
	return nil
}

// MarshalJSON will marshal a config to JSON.
func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Builder)
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

	builderFunc, ok := registry[typeString]
	if !ok {
		return fmt.Errorf("unsupported type '%s'", typeString)
	}

	builder := builderFunc()
	err = unmarshal(builder)
	if err != nil {
		return fmt.Errorf("unmarshal to %s: %s", typeString, err)
	}

	c.Builder = builder
	return nil
}

// MarshalYAML will marshal a config to YAML.
func (c Config) MarshalYAML() (interface{}, error) {
	return c.Builder, nil
}
