package plugin

import (
	"fmt"
)

func init() {
	RegisterConfig("bundle", &BundleConfig{})
}

type BundleConfig struct {
	DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
	DefaultInputterConfig  `mapstructure:",squash" yaml:",inline"`
	DefaultBundleConfig    `mapstructure:",squash" yaml:",inline"`
}

func (c BundleConfig) Build(context BuildContext) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	configs, err := c.DefaultBundleConfig.RenderPluginConfigs(context.Bundles)
	if err != nil {
		return nil, fmt.Errorf("render bundle configs: %s", err)
	}

	isInputter := hasBundleOfType(configs, "bundle_input")
	isOutputter := hasBundleOfType(configs, "bundle_output")

	var defaultInputter DefaultInputter
	var defaultOutputter DefaultOutputter
	if isInputter {
		defaultInputter, err = c.DefaultInputterConfig.Build()
		context.BundleInput = defaultInputter.Input()
	}
	if err != nil {
		return nil, fmt.Errorf("build default inputter: %s", err)
	}

	if isOutputter {
		defaultOutputter, err = c.DefaultOutputterConfig.Build(context.Plugins)
		context.BundleOutput = defaultOutputter.Output()
	}
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	defaultBundle, err := c.DefaultBundleConfig.Build(configs, context)
	if err != nil {
		return nil, fmt.Errorf("build default bundle: %s", err)
	}

	var plugin Plugin
	switch {
	case isInputter && isOutputter:
		plugin = &BothputterBundle{
			DefaultPlugin:    defaultPlugin,
			DefaultBundle:    defaultBundle,
			DefaultInputter:  defaultInputter,
			DefaultOutputter: defaultOutputter,
		}
	case isInputter && !isOutputter:
		plugin = &InputterBundle{
			DefaultPlugin:   defaultPlugin,
			DefaultBundle:   defaultBundle,
			DefaultInputter: defaultInputter,
		}
	case !isInputter && isOutputter:
		plugin = &OutputterBundle{
			DefaultPlugin:    defaultPlugin,
			DefaultBundle:    defaultBundle,
			DefaultOutputter: defaultOutputter,
		}
	case isInputter && !isOutputter:
		plugin = &NeitherputterBundle{
			DefaultPlugin: defaultPlugin,
			DefaultBundle: defaultBundle,
		}
	}

	return plugin, nil
}

func hasBundleOfType(configs []PluginConfig, bundleType string) bool {
	for _, config := range configs {
		if config.Type() == bundleType {
			return true
		}
	}
	return false
}

type InputterBundle struct {
	DefaultPlugin
	DefaultBundle
	DefaultInputter
}
type OutputterBundle struct {
	DefaultPlugin
	DefaultBundle
	DefaultOutputter
}

type NeitherputterBundle struct {
	DefaultPlugin
	DefaultBundle
}

type BothputterBundle struct {
	DefaultPlugin
	DefaultBundle
	DefaultInputter
	DefaultOutputter
}
