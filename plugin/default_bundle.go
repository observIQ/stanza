package plugin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type DefaultBundleConfig struct {
	BundleType string `mapstructure:"bundle_type"`
	Params     map[string]interface{}
}
Config
func (c DefaultBundleConfig) RenderPluginConfigs(bundles []*bundle.BundleDefinition) ([]PluginConfig, error) {
	var bundleDefinition *bundle.BundleDefinition
	for _, bundle := range bundles {
		if c.BundleType == bundle.BundleType {
			bundleDefinition = bundle
			break // TODO warn on duplicate
		}
	}
	if bundleDefinition == nil {
		return nil, fmt.Errorf("bundle definition with type %s not found in bundle path", c.BundleType)
	}

	// Render the bundle config
	renderedConfig, err := bundleDefinition.Render(c.Params)
	if err != nil {
		return nil, fmt.Errorf("render bundle config: %s", err)
	}

	// Parse the rendered config
	// TODO reuse this code
	v := viper.New()
	v.SetConfigType("yaml")
	err = v.ReadConfig(renderedConfig)
	if err != nil {
		return nil, fmt.Errorf("read config into viper: %s", err)
	}
	var pluginUConfigstruct {
		Plugins []PluginConfig
	}
	err = v.UnmarshalExact(&pluginUnmarshaller, func(c *mapstructure.DecoderConfig) {
		c.DecodeHook = PluginConfigDecoder
	})
	if err != nil {
		return nil, fmt.Errorf("unmarshal from viper: %s", err)
	}

	return pluginUnmarshaller.Plugins, nil
}
