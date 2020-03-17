package plugin

import (
	"fmt"
	"reflect"

	// Register built-in plugins
	"github.com/bluemedora/bplogagent/bundle"
	"github.com/mitchellh/mapstructure"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

var PluginConfigDefinitions = make(map[string]func() PluginConfig)

// RegisterConfig will register a config struct by name in the packages config registry
// during package load time.
func RegisterConfig(name string, config PluginConfig) {
	PluginConfigDefinitions[name] = func() PluginConfig {
		val := reflect.New(reflect.TypeOf(config).Elem()).Interface()
		return val.(PluginConfig)
	}
}

type PluginConfig interface {
	ID() PluginID
	Type() string
	Build(BuildContext) (Plugin, error)
}

type OutputterConfig interface {
	PluginConfig
	OutputIDs() []PluginID
}

type InputterConfig interface {
	PluginConfig
	IsInputter()
}

type BuildContext struct {
	Plugins map[PluginID]Plugin
	Bundles []*bundle.BundleDefinition
	// TODO this should be an array of bundle IDs to namespace the plugin ids in the bundles
	IsBundle bool
	Database *bbolt.DB
	Logger   *zap.SugaredLogger
}

func UnmarshalHook(c *mapstructure.DecoderConfig) {
	c.DecodeHook = newPluginConfigDecoder()
}

func newPluginConfigDecoder() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		var m map[interface{}]interface{}
		if f != reflect.TypeOf(m) {
			return data, nil
		}

		if t.String() != "plugin.PluginConfig" {
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

		configDefinitionFunc, ok := PluginConfigDefinitions[typeString]
		if !ok {
			return nil, fmt.Errorf("unknown plugin config type %s", typeString)
		}

		configDefinition := configDefinitionFunc()
		// TODO handle unused keys
		err := mapstructure.Decode(data, &configDefinition)
		if err != nil {
			return nil, fmt.Errorf("decode plugin definition: %s", err)
		}

		return configDefinition, nil
	}
}
