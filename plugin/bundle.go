package plugin

func init() {
	RegisterConfig("inputter_bundle", &InputterBundleConfig{})
	RegisterConfig("outputter_bundle", &OutputterBundleConfig{})
	RegisterConfig("bothputter_bundle", &BothputterBundleConfig{})
	RegisterConfig("neitherputter_bundle", &NeitherputterBundleConfig{})
}

type DefaultBundleConfig struct {
	BundleType    string `mapstructure:"bundle_type"`
	PluginConfigs []PluginConfig
}

type InputterBundleConfig struct {
	DefaultPluginConfig   `mapstructure:",squash"`
	DefaultInputterConfig `mapstructure:",squash"`
	DefaultBundleConfig   `mapstructure:",squash"`
}

type OutputterBundleConfig struct {
	DefaultPluginConfig    `mapstructure:",squash"`
	DefaultOutputterConfig `mapstructure:",squash"`
	DefaultBundleConfig    `mapstructure:",squash"`
}

type BothputterBundleConfig struct {
	DefaultPluginConfig    `mapstructure:",squash"`
	DefaultOutputterConfig `mapstructure:",squash"`
	DefaultInputterConfig  `mapstructure:",squash"`
	DefaultBundleConfig    `mapstructure:",squash"`
}

type NeitherputterBundleConfig struct {
	DefaultPluginConfig `mapstructure:",squash"`
	DefaultBundleConfig `mapstructure:",squash"`
}
