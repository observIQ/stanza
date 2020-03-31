package plugin

import (
	"fmt"
	"reflect"

	"github.com/bluemedora/bplogagent/bundle"
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
	plugins := make([]Plugin, len(configs))

	for _, config := range configs {
		plugin, err := config.Build(context)
		if err != nil {
			return plugins, fmt.Errorf("failed to build %s: %s", config.ID(), err)
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// configDefinitions is a registry of plugin types to plugin configs.
var configDefinitions = make(map[string]func() Config)

// Register will register a plugin config by plugin type.
func Register(pluginType string, config Config) {
	configDefinitions[pluginType] = func() Config {
		val := reflect.New(reflect.TypeOf(config).Elem()).Interface()
		return val.(Config)
	}
}

// ConfigDecoder is a function that uses the config registry to unmarshal plugin configs.
var ConfigDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	var m map[interface{}]interface{}
	if f != reflect.TypeOf(m) {
		return data, nil
	}

	if t.String() != "plugin.Config" {
		return data, nil
	}

	d, ok := data.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data type %T for plugin config", data)
	}

	typeInterface, ok := d["type"]
	if !ok {
		return nil, fmt.Errorf("missing type field for plugin config")
	}

	typeString, ok := typeInterface.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for plugin config type", typeInterface)
	}

	createConfig, ok := configDefinitions[typeString]
	if !ok {
		return nil, fmt.Errorf("unknown plugin config type %s", typeString)
	}

	config := createConfig()
	// TODO handle unused keys
	err := mapstructure.Decode(data, &config)
	if err != nil {
		return nil, fmt.Errorf("decode plugin definition: %s", err)
	}

	return config, nil
}
