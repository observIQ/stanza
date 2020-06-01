package pipeline

import (
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/custom"
	yaml "gopkg.in/yaml.v2"
)

// Config is the configuration of a pipeline.
type Config []Params

// BuildPipeline will build a pipeline from the config.
func (c Config) BuildPipeline(context plugin.BuildContext) (*Pipeline, error) {
	pluginConfigs, err := c.buildPluginConfigs()
	if err != nil {
		return nil, errors.Wrap(err, "build plugin configs")
	}

	plugins, err := c.buildPlugins(pluginConfigs, context)
	if err != nil {
		return nil, errors.Wrap(err, "build plugins")
	}

	pipeline, err := NewPipeline(plugins)
	if err != nil {
		return nil, errors.Wrap(err, "new pipeline")
	}

	return pipeline, nil
}

func (c Config) buildPlugins(pluginConfigs []plugin.Config, context plugin.BuildContext) ([]plugin.Plugin, error) {
	plugins := make([]plugin.Plugin, 0, len(pluginConfigs))
	for _, pluginConfig := range pluginConfigs {
		plugin, err := pluginConfig.Build(context)

		if err != nil {
			return nil, errors.WithDetails(err, "plugin_id", pluginConfig.ID(), "plugin_type", pluginConfig.Type())
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func (c Config) buildPluginConfigs() ([]plugin.Config, error) {
	pluginConfigs := make([]plugin.Config, 0, len(c))

	for _, params := range c {
		if err := params.Validate(); err != nil {
			return nil, errors.Wrap(err, "validate config params")
		}

		if params.IsBuiltin() {
			pluginConfig, err := plugin.BuildConfig(params, "$")
			if err != nil {
				return nil, errors.Wrap(err, "parse builtin plugin config")
			}
			pluginConfigs = append(pluginConfigs, pluginConfig)
		}

		if params.IsCustom() {
			customConfig, err := custom.BuildConfig(params, "$")
			if err != nil {
				return nil, errors.Wrap(err, "parse custom plugin config")
			}
			pluginConfigs = append(pluginConfigs, customConfig.Pipeline...)
		}
	}

	return pluginConfigs, nil
}

// Params is a raw params map that can be converted into a plugin config.
type Params map[string]interface{}

// ID returns the id field in the params map.
func (p Params) ID() string {
	return p.getString("id")
}

// Type returns the type field in the params map.
func (p Params) Type() string {
	return p.getString("type")
}

// IsBuiltin will return a boolean indicating if the params represent a builtin plugin.
func (p Params) IsBuiltin() bool {
	return plugin.IsDefined(p.Type())
}

// IsCustom will return a boolean indicating if the params represent a custom plugin.
func (p Params) IsCustom() bool {
	return custom.IsDefined(p.Type())
}

// String will return the string representation of the params
func (p Params) String() string {
	bytes, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}

	return string(bytes)
}

// Validate will validate the basic fields required to make a plugin config.
func (p Params) Validate() error {
	if p.ID() == "" {
		return errors.NewError(
			"missing required `id` field for plugin config",
			"ensure that all plugin configs have a defined id field",
			"config", p.String(),
		)
	}

	if p.Type() == "" {
		return errors.NewError(
			"missing required `type` field for plugin config",
			"ensure that all plugin configs have a defined type field",
			"config", p.String(),
		)
	}

	if !p.IsBuiltin() && !p.IsCustom() {
		return errors.NewError(
			"unsupported `type` field for plugin config",
			"ensure that all plugins have a supported builtin or custom type",
			"config", p.String(),
		)
	}

	return nil
}

// getString returns a string value from the params block
func (p Params) getString(key string) string {
	rawValue, ok := p[key]
	if !ok {
		return ""
	}

	stringValue, ok := rawValue.(string)
	if !ok {
		return ""
	}

	return stringValue
}
