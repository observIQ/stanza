package plugin

import (
	"testing"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestBasicBundlePluginFunctionality(t *testing.T) {
	config := &BundleConfig{
		DefaultPluginConfig: DefaultPluginConfig{
			PluginID:   "mybundle",
			PluginType: "bundle",
		},
		DefaultBundleConfig: DefaultBundleConfig{
			BundleType: "noop",
			Params: map[string]interface{}{
				"enabled": true,
			},
		},
		DefaultOutputterConfig: DefaultOutputterConfig{
			Output: "mybundlereceiver",
		},
	}

	logger, err := zap.NewProduction()
	assert.NoError(t, err)

	bundles := bundle.GetBundleDefinitions("./test/bundles", logger.Sugar())
	assert.Greater(t, len(bundles), 0)

	buildContext := BuildContext{
		Plugins: map[PluginID]Plugin{
			"mybundlereceiver": &NullOutput{
				DefaultPlugin: DefaultPlugin{
					id:            "mybundlereceiver",
					pluginType:    "null",
					SugaredLogger: logger.Sugar(),
				},
				DefaultInputter: DefaultInputter{
					input: make(EntryChannel, 1),
				},
			},
		},
		Bundles: bundles,
		Logger:  logger.Sugar(),
	}

	_, err = config.Build(buildContext)
	assert.NoError(t, err)
}
