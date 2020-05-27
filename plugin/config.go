package plugin

import (
	"encoding/json"
	"reflect"

	"github.com/bluemedora/bplogagent/errors"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

type Config struct {
	PluginBuilder `yaml:",inline"`
}

type PluginBuilder interface {
	ID() string
	Type() string
	Build(BuildContext) (Plugin, error)
}

// BuildContext supplies contextual resources when building a plugin.
type BuildContext struct {
	Database *bbolt.DB
	Logger   *zap.SugaredLogger
}

// BuildPlugins will build a collection of plugins from plugin configs.
func BuildPlugins(configs []Config, context BuildContext) ([]Plugin, error) {
	plugins := make([]Plugin, 0, len(configs))

	for _, config := range configs {
		plugin, err := config.Build(context)
		if err != nil {
			return plugins, errors.WithDetails(err,
				"plugin_id", config.ID(),
				"plugin_type", config.Type(),
			)
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// configDefinitions is a registry of plugin types to plugin configs.
var configDefinitions = make(map[string]func() interface{})

// Register will register a plugin config by plugin type.
func Register(pluginType string, config PluginBuilder) {
	configDefinitions[pluginType] = func() interface{} {
		val := reflect.New(reflect.TypeOf(config).Elem()).Interface()
		return val.(PluginBuilder)
	}
}

func (c *Config) UnmarshalJSON(raw []byte) error {
	var typeDecoder struct {
		Type string
	}
	err := json.Unmarshal(raw, &typeDecoder)
	if err != nil {
		return err
	}

	if typeDecoder.Type == "" {
		return ErrMissingType
	}

	pluginBuilderGenerator, ok := configDefinitions[typeDecoder.Type]
	if !ok {
		return NewErrUnknownType(typeDecoder.Type)
	}

	pluginBuilder := pluginBuilderGenerator()
	err = json.Unmarshal(raw, pluginBuilder)
	if err != nil {
		return err
	}

	c.PluginBuilder = pluginBuilder.(PluginBuilder)
	return nil
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.PluginBuilder)
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var typeDecoder struct {
		Type string
	}
	err := unmarshal(&typeDecoder)
	if err != nil {
		return err
	}

	if typeDecoder.Type == "" {
		return ErrMissingType
	}

	pluginBuilderGenerator, ok := configDefinitions[typeDecoder.Type]
	if !ok {
		return NewErrUnknownType(typeDecoder.Type)
	}

	pluginBuilder := pluginBuilderGenerator()
	err = unmarshal(pluginBuilder)
	if err != nil {
		return err
	}

	c.PluginBuilder = pluginBuilder.(PluginBuilder)
	return nil
}

func (c Config) MarshalYAML() (interface{}, error) {
	return c.PluginBuilder, nil
}

/*********
  Errors
*********/

var ErrMissingType = errors.NewError(
	"Missing required field `type`.",
	"Ensure that all plugin configs have a `type` field set",
)

func NewErrUnknownType(pluginType string) errors.AgentError {
	return errors.NewError(
		"Plugin config has an unknown plugin type.",
		"Ensure that all plugin configs have a known, valid type.",
		"plugin_type", pluginType,
	)
}
