package plugin

import (
	"reflect"

	"github.com/bluemedora/log-agent/config"
)

var ConfigDefinitions = make(map[string]func() interface{})

func registerConfig(name string, configExample interface{}) {
	if _, ok := configExample.(config.PluginConfig); ok {
		ConfigDefinitions[name] = func() interface{} {
			return reflect.ValueOf(config).Interface()
		}
	}
}
