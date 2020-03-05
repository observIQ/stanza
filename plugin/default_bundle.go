package plugin

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type DefaultBundleConfig struct {
	BundleType string `mapstructure:"bundle_type"`
	Params     map[string]interface{}
}

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
		return nil, fmt.Errorf("failed to render bundle config: %s", err)
	}

	// Parse the rendered config
	// TODO reuse this code
	v := viper.New()
	v.SetConfigType("yaml")
	err = v.ReadConfig(renderedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read config into viper: %s", err)
	}
	var pluginUnmarshaller struct {
		Plugins []PluginConfig
	}
	err = v.UnmarshalExact(&pluginUnmarshaller, UnmarshalHook)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal from viper: %s", err)
	}

	return pluginUnmarshaller.Plugins, nil
}

func (c DefaultBundleConfig) Build(configs []PluginConfig, context BuildContext) (DefaultBundle, error) {
	// Clear plugins before build
	context.Plugins = make(map[PluginID]Plugin)
	plugins, err := BuildPlugins(configs, context)
	if err != nil {
		return DefaultBundle{}, fmt.Errorf("failed to build bundle plugins: %s", err)
	}

	defaultBundle := DefaultBundle{
		bundleType:    c.BundleType,
		plugins:       plugins,
		SugaredLogger: context.Logger,
	}

	return defaultBundle, nil
}

type DefaultBundle struct {
	bundleType string
	plugins    []Plugin
	pluginWg   *sync.WaitGroup

	*zap.SugaredLogger
}

func (b *DefaultBundle) Start(wg *sync.WaitGroup) error {
	// TODO
	ready := make(chan error)
	go func() {
		defer wg.Done()
		pluginWg := &sync.WaitGroup{}
		err := StartPlugins(b.plugins, pluginWg, b.SugaredLogger)
		if err != nil {
			ready <- fmt.Errorf("failed to start bundle plugins: %s", err)
			// TODO stop plugins if errored?
			return
		}

		ready <- nil
		pluginWg.Wait()
	}()

	return <-ready
}

func (b *DefaultBundle) Stop() {
	for _, plugin := range b.plugins {
		if stopper, ok := plugin.(Stopper); ok {
			stopper.Stop()
		}
	}
}
