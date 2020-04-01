package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestBasicBundlePluginFunctionality(t *testing.T) {
	config := &BundleConfig{
		PluginConfig: base.PluginConfig{
			PluginID:   "mybundle",
			PluginType: "bundle",
		},
		OutputID:   "mybundlereceiver",
		BundleType: "noop",
		Params: map[string]interface{}{
			"enabled": true,
		},
	}

	logger, err := zap.NewProduction()
	assert.NoError(t, err)

	bundles := bundle.GetBundleDefinitions("./test/bundles", logger.Sugar())
	assert.Greater(t, len(bundles), 0)

	buildContext := plugin.BuildContext{
		Bundles: bundles,
		Logger:  logger.Sugar(),
	}

	_, err = config.Build(buildContext)
	assert.NoError(t, err)
}
