package plugin

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/mitchellh/mapstructure"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

type Config struct {
	PluginBuilder `mapstructure:",squash"`
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
var configDefinitions = make(map[string]func() (interface{}, mapstructure.DecodeHookFunc))

// Register will register a plugin config by plugin type.
func Register(pluginType string, config PluginBuilder, decoders ...mapstructure.DecodeHookFunc) {
	configDefinitions[pluginType] = func() (interface{}, mapstructure.DecodeHookFunc) {
		val := reflect.New(reflect.TypeOf(config).Elem()).Interface()
		return val.(PluginBuilder), mapstructure.ComposeDecodeHookFunc(decoders...)
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

	pluginBuilder, _ := pluginBuilderGenerator()
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

	pluginBuilder, _ := pluginBuilderGenerator()
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

// ConfigDecoder is a function that uses the config registry to unmarshal plugin configs.
var ConfigDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t.String() != "plugin.Config" {
		return data, nil
	}

	var mapInterface map[interface{}]interface{}
	var mapString map[string]interface{}
	switch f {
	case reflect.TypeOf(mapInterface):
		mapString = make(map[string]interface{})
		for k, v := range data.(map[interface{}]interface{}) {
			if kString, ok := k.(string); ok {
				mapString[kString] = v
			} else {
				return nil, fmt.Errorf("map has non-string key")
			}
		}
	case reflect.TypeOf(mapString):
		mapString = data.(map[string]interface{})
	default:
		return data, nil
	}

	typeInterface, ok := mapString["type"]
	if !ok {
		return nil, ErrMissingType
	}

	typeString, ok := typeInterface.(string)
	if !ok {
		return nil, errors.NewError(
			"Plugin config does not have a `type` field as a string.",
			"Ensure that all plugin configs have a `type` field formatted as a string.",
		)
	}

	createConfig, ok := configDefinitions[typeString]
	if !ok {
		return nil, NewErrUnknownType(typeString)
	}

	config, decodeHook := createConfig()
	decoderCfg := &mapstructure.DecoderConfig{
		Result:     &config,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(decodeHook, entry.FieldSelectorDecoder),
	}
	decoder, err := mapstructure.NewDecoder(decoderCfg)
	if err != nil {
		return nil, fmt.Errorf("build decoder: %w", err)
	}

	err = decoder.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode plugin definition: %s", err)
	}

	return &Config{config.(PluginBuilder)}, nil
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
