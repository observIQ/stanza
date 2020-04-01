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
	base.PluginConfig `mapstructure:",squash" yaml:",inline"`
	BundleType        string `mapstructure:"bundle_type" yaml:"bundle_type"`
	Params            map[string]interface{}
	OutputID          string `mapstructure:"output" yaml:"output"`
}

// Build will build a bundle plugin.
func (c BundleConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return nil, err
	}

	configs, err := c.renderPluginConfigs(context.Bundles)
	if err != nil {
		return nil, fmt.Errorf("bundle failed to render plugin configs: %s", err)
	}

	plugins, err := plugin.BuildPlugins(configs, context)
	if err != nil {
		return nil, fmt.Errorf("bundle failed to build plugins: %s", err)
	}

	bundleInput := findBundleInput(plugins)
	bundleOutput := findBundleOutput(plugins)

	if bundleOutput == nil && c.OutputID != "" {
		return nil, fmt.Errorf("bundle has an output param, but no bundle_output plugin")
	}

	if bundleOutput != nil && c.OutputID == "" {
		return nil, fmt.Errorf("bundle has a bundle_output plugin, but no output param")
	}

	bundle := &Bundle{
		Plugin:       p,
		OutputID:     c.OutputID,
		plugins:      plugins,
		bundleInput:  bundleInput,
		bundleOutput: bundleOutput,
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

// findBundleInputs will find the first bundle input in a collection of plugins.
func findBundleInput(plugins []plugin.Plugin) *BundleInput {
	for _, plugin := range plugins {
		switch t := plugin.(type) {
		case *BundleInput:
			return t
		}
	}
	return nil
}

// findBundleOutput will find the first bundle output in a collection of plugins.
func findBundleOutput(plugins []plugin.Plugin) *BundleOutput {
	for _, plugin := range plugins {
		switch t := plugin.(type) {
		case *BundleOutput:
			return t
		}
	}
	return nil
}

// Bundle is a plugin that runs its own collection of plugins in a pipeline.
type Bundle struct {
	base.Plugin
	OutputID string
	Output   plugin.Consumer

	pipeline     *pipeline.Pipeline
	plugins      []plugin.Plugin
	bundleInput  *BundleInput
	bundleOutput *BundleOutput
}

// Start will start the bundle pipeline.
func (b *Bundle) Start() error {
	pipeline, err := pipeline.NewPipeline(b.plugins)
	if err != nil {
		return fmt.Errorf("build pipeline: %s", err)
	}
	b.pipeline = pipeline

	if b.bundleOutput != nil {
		b.bundleOutput.SetBundle(b)
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

// Consumers will return an array containing the plugin's output, if one exists.
func (b *Bundle) Consumers() []plugin.Consumer {
	if b.Output != nil {
		return []plugin.Consumer{}
	}

	return []plugin.Consumer{b.Output}
}

// SetConsumers will find an output consumer if output id is not empty.
func (b *Bundle) SetConsumers(consumers []plugin.Consumer) error {
	if b.OutputID == "" {
		return nil
	}

	consumer, err := base.FindConsumer(consumers, b.OutputID)
	if err != nil {
		return err
	}

	b.Output = consumer
	return nil
}

// Consume will send an entry to the pipeline.
func (b *Bundle) Consume(entry *entry.Entry) error {
	if b.bundleInput == nil {
		return fmt.Errorf("bundle_input plugin does not exist")
	}

	return b.bundleInput.PipeIn(entry)
}
