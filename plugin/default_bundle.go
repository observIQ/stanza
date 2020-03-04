package plugin

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

type DefaultBundleConfig struct {
	BundleType    string `mapstructure:"bundle_type"`
	PluginConfigs []PluginConfig
}

func (c DefaultBundleConfig) Build(context BuildContext) (DefaultBundle, error) {
	plugins := make([]Plugin, len(c.PluginConfigs))
	for i, config := range c.PluginConfigs {
		plugin, err := config.Build(context)
		if err != nil {
			return DefaultBundle{}, fmt.Errorf("failed to build bundle plugin: %s", err)
		}
		plugins[i] = plugin
	}

	plugin := DefaultBundle{
		bundleType: c.BundleType,
		plugins:    plugins,
		// TODO ensure that loggers are namspaced correctly everywhere
		SugaredLogger: context.Logger,
	}
	return plugin, nil
}

type DefaultBundle struct {
	bundleType string
	plugins    []Plugin
	pluginWg   *sync.WaitGroup

	*zap.SugaredLogger
}

func (b *DefaultBundle) Start(wg *sync.WaitGroup) error {
	// TODO
	go func() {
		defer wg.Done()
	}()
	return nil
}
