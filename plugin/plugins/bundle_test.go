package plugins

import (
	"testing"

	"github.com/bluemedora/bplogagent/bundle"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestBasicBundlePluginFunctionality(t *testing.T) {
	config := &BundleConfig{
		DefaultPluginConfig: pg.DefaultPluginConfig{
			PluginID:   "mybundle",
			PluginType: "bundle",
		},
		DefaultOutputterConfig: pg.DefaultOutputterConfig{
			Output: "mybundlereceiver",
		},
		BundleType: "noop",
		Params: map[string]interface{}{
			"enabled": true,
		},
	}

	logger, err := zap.NewProduction()
	assert.NoError(t, err)

	bundles := bundle.GetBundleDefinitions("./test/bundles", logger.Sugar())
	assert.Greater(t, len(bundles), 0)

	buildContext := pg.BuildContext{
		Plugins: map[pg.PluginID]pg.Plugin{
			"mybundlereceiver": &DropOutput{
				DefaultPlugin: pg.DefaultPlugin{
					PluginID:      "mybundlereceiver",
					PluginType:    "null",
					SugaredLogger: logger.Sugar(),
				},
			},
		},
		Bundles: bundles,
		Logger:  logger.Sugar(),
	}

	_, err = config.Build(buildContext)
	assert.NoError(t, err)
}
