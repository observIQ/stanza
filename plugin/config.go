package plugin

import (
	"fmt"
	"reflect"

	"github.com/awalterschulze/gographviz"
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
	ID() PluginID
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

func BuildPlugins(configs []PluginConfig, logger *zap.SugaredLogger) ([]Plugin, error) {
	plugins := make([]Plugin, 0, len(configs))
	for _, config := range configs {
		plugin, err := config.Build(logger)
		if err != nil {
			return nil, fmt.Errorf("failed to build plugin with ID '%s': %s", config.ID(), err)
		}

		plugins = append(plugins, plugin)
	}

	err := setPluginOutputs(plugins, logger)
	if err != nil {
		return nil, err
	}

	return plugins, nil
}

func setPluginOutputs(plugins []Plugin, logger *zap.SugaredLogger) error {
	processorInputs := make(map[PluginID]EntryChannel)
	graphAst, _ := gographviz.ParseString(`digraph G {}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		panic(err)
	}

	// Generate the list of input channels
	for _, plugin := range plugins {
		if inputter, ok := plugin.(Inputter); ok {
			// TODO check for duplicate IDs
			processorInputs[plugin.ID()] = inputter.Input()
		}
		graph.AddNode("G", string(plugin.ID()), nil)
	}

	// Set the output channels using the generated lists
	for _, plugin := range plugins {
		if outputter, ok := plugin.(Outputter); ok {
			err := outputter.SetOutputs(processorInputs)
			if err != nil {
				return fmt.Errorf("failed to set outputs for plugin with ID %v: %s", plugin.ID(), err)
			}

			for id, _ := range outputter.Outputs() {
				graph.AddEdge(string(plugin.ID()), string(id), true, nil)
			}
		}
	}

	logger.Infof("Generated graphviz chart: %s", graph)

	return nil
}
