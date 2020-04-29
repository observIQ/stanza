package plugin

import (
	"fmt"
	"reflect"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/mitchellh/mapstructure"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// Config defines the configuration and build process of a plugin.
type Config interface {
	ID() string
	Type() string
	Build(BuildContext) (Plugin, error)
}

// BuildContext supplies contextual resources when building a plugin.
type BuildContext struct {
	Bundles  []*bundle.BundleDefinition
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
var configDefinitions = make(map[string]func() (Config, mapstructure.DecodeHookFunc))

// Register will register a plugin config by plugin type.
func Register(pluginType string, config Config, decoders ...mapstructure.DecodeHookFunc) {
	configDefinitions[pluginType] = func() (Config, mapstructure.DecodeHookFunc) {
		val := reflect.New(reflect.TypeOf(config).Elem()).Interface()
		return val.(Config), mapstructure.ComposeDecodeHookFunc(decoders...)
	}
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
		return nil, errors.NewError(
			"Plugin config is missing a `type` field.",
			"Ensure that all plugin configs have a `type` field defined.",
		)
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
		return nil, errors.NewError(
			"Plugin config has an unknown plugin type.",
			"Ensure that all plugin configs have a known, valid type.",
			"plugin_type", typeString,
		)
	}

	config, decodeHook := createConfig()
	// TODO handle unused fields
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

	return config, nil
}
