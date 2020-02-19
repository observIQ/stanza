package plugin

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type PluginConfig interface{}

var PluginConfigDefinitions = make(map[string]func() interface{})

func RegisterConfig(name string, config interface{}) {
	if _, ok := config.(PluginConfig); ok {
		PluginConfigDefinitions[name] = func() interface{} {
			return reflect.ValueOf(config).Interface()
		}
	}
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
		err := mapstructure.Decode(data, &configDefinition)
		if err != nil {
			return nil, fmt.Errorf("failed to decode plugin definition: %s", err)
		}

		return configDefinition, nil
	}
}
