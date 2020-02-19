package plugin

import "reflect"

type PluginConfig interface{}

var PluginConfigDefinitions = make(map[string]func() interface{})

func RegisterConfig(name string, config interface{}) {
	if _, ok := config.(PluginConfig); ok {
		PluginConfigDefinitions[name] = func() interface{} {
			return reflect.ValueOf(config).Interface()
		}
	}
}
