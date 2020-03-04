package plugin

import "fmt"

func init() {
	RegisterConfig("inputter_bundle", &InputterBundleConfig{})
}

type InputterBundleConfig struct {
	DefaultPluginConfig   `mapstructure:",squash"`
	DefaultInputterConfig `mapstructure:",squash"`
	DefaultBundleConfig   `mapstructure:",squash"`
}

func (c InputterBundleConfig) Build(context BuildContext) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	defaultBundle, err := c.DefaultBundleConfig.Build(context)
	if err != nil {
		return nil, fmt.Errorf("failed to build default bundle: %s", err)
	}

	plugin := &InputterBundle{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,
		DefaultBundle:   defaultBundle,
	}

	return plugin, nil
}

type InputterBundle struct {
	DefaultPlugin
	DefaultInputter
	DefaultBundle
}
