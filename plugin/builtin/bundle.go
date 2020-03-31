package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/pipeline"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func init() {
	plugin.Register("bundle", &BundleConfig{})
}

// BundleConfig is the configuration of a bundle plugin.
type BundleConfig struct {
	base.InputConfig `mapstructure:",squash" yaml:",inline"`
	BundleType       string `mapstructure:"bundle_type" yaml:"bundle_type"`
	Params           map[string]interface{}
}

// Build will build a bundle plugin.
func (c BundleConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	configs, err := c.renderPluginConfigs(context.Bundles)
	if err != nil {
		return nil, fmt.Errorf("render bundle configs: %s", err)
	}

	plugins, err := plugin.BuildPlugins(configs, context)
	if err != nil {
		return nil, fmt.Errorf("plugins failed to build in bundle: %s", err)
	}

	bundle := &Bundle{
		InputPlugin:   inputPlugin,
		plugins:       plugins,
		bundleInputs:  findBundleInputs(plugins),
		bundleOutputs: findBundleOutputs(plugins),
	}

	return bundle, nil
}

func (c BundleConfig) renderPluginConfigs(bundles []*bundle.BundleDefinition) ([]plugin.Config, error) {
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
	var pluginUnmarshaller struct {
		Plugins []plugin.Config
	}
	err = v.UnmarshalExact(&pluginUnmarshaller, func(c *mapstructure.DecoderConfig) {
		c.DecodeHook = plugin.ConfigDecoder
	})
	if err != nil {
		return nil, fmt.Errorf("unmarshal from viper: %s", err)
	}

	return pluginUnmarshaller.Plugins, nil
}

// findBundleInputs will find all plugins that are of type BundleInput.
func findBundleInputs(plugins []plugin.Plugin) []*BundleInput {
	bundleInputs := make([]*BundleInput, 0)
	for _, plugin := range plugins {
		switch c := plugin.(type) {
		case *BundleInput:
			bundleInputs = append(bundleInputs, c)
		}
	}
	return bundleInputs
}

// findBundleOutputs will find all plugins that are of type BundleOutput.
func findBundleOutputs(plugins []plugin.Plugin) []*BundleOutput {
	bundleOutputs := make([]*BundleOutput, 0)
	for _, plugin := range plugins {
		switch c := plugin.(type) {
		case *BundleOutput:
			bundleOutputs = append(bundleOutputs, c)
		}
	}
	return bundleOutputs
}

// Bundle is a plugin that runs its own collection of plugins in a pipeline.
type Bundle struct {
	base.InputPlugin
	pipeline      *pipeline.Pipeline
	plugins       []plugin.Plugin
	bundleInputs  []*BundleInput
	bundleOutputs []*BundleOutput
}

// Start will start the bundle pipeline.
func (b *Bundle) Start() error {
	pipeline, err := pipeline.NewPipeline(b.plugins)
	if err != nil {
		return fmt.Errorf("build pipeline: %s", err)
	}
	b.pipeline = pipeline

	for _, bundleOutput := range b.bundleOutputs {
		bundleOutput.SetBundle(b)
	}

	err = b.pipeline.Start()
	if err != nil {
		return fmt.Errorf("start bundle pipeline: %s", err)
	}

	return nil
}

// Stop will stop the bundle pipeline.
func (b *Bundle) Stop() error {
	b.pipeline.Stop()
	b.pipeline = nil
	return nil
}

// PipelineOut will forward an outgoing entry from the pipeline.
func (b *Bundle) PipelineOut(entry *entry.Entry) error {
	return b.Output.Consume(entry)
}

// Consume will send an entry to the pipeline.
func (b *Bundle) Consume(entry *entry.Entry) error {
	for _, bundleInput := range b.bundleInputs {
		if err := bundleInput.Consume(entry); err != nil {
			return err
		}
	}
	return nil
}
