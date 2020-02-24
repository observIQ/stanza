package plugin

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

var PluginConfigDefinitions = make(map[string]func() PluginConfig)

// RegisterConfig will register a config struct by name in the packages config registry
// during package load time.
func RegisterConfig(name string, config interface{}) {
	if _, ok := config.(PluginConfig); ok {
		PluginConfigDefinitions[name] = func() PluginConfig {
			return reflect.ValueOf(config).Interface().(PluginConfig)
		}
	} else {
		panic(fmt.Sprintf("plugin type %v does not implement the plugin.PluginConfig interface", name))
	}
}

type PluginConfig interface {
	Build(*zap.SugaredLogger) (Plugin, error)
	ID() string
}

func UnmarshalHook(c *mapstructure.DecoderConfig) {
	c.DecodeHook = PluginConfigToStructHookFunc()
}

func PluginConfigToStructHookFunc() mapstructure.DecodeHookFunc {
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
			return nil, fmt.Errorf("failed to decode plugin definition: %s", err)
		}

		return configDefinition, nil
	}
}
