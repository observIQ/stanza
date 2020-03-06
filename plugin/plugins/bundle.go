package plugins

import (
	"fmt"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("bundle", &BundleConfig{})
}

type BundleConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig  `mapstructure:",squash" yaml:",inline"`
	pg.DefaultBundleConfig    `mapstructure:",squash" yaml:",inline"`
}

func (c BundleConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
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

	var defaultInputter pg.DefaultInputter
	var defaultOutputter pg.DefaultOutputter
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

	var plugin pg.Plugin
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

func hasBundleOfType(configs []pg.PluginConfig, bundleType string) bool {
	for _, config := range configs {
		if config.Type() == bundleType {
			return true
		}
	}
	return false
}

type InputterBundle struct {
	pg.DefaultPlugin
	pg.DefaultBundle
	pg.DefaultInputter
}
type OutputterBundle struct {
	pg.DefaultPlugin
	pg.DefaultBundle
	pg.DefaultOutputter
}

type NeitherputterBundle struct {
	pg.DefaultPlugin
	pg.DefaultBundle
}

type BothputterBundle struct {
	pg.DefaultPlugin
	pg.DefaultBundle
	pg.DefaultInputter
	pg.DefaultOutputter
}
