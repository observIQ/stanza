package helper

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasicConfigID(t *testing.T) {
	config := BasicConfig{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	require.Equal(t, "test-id", config.ID())
}

func TestBasicConfigType(t *testing.T) {
	config := BasicConfig{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	require.Equal(t, "test-type", config.Type())
}

func TestBasicConfigBuildWithoutID(t *testing.T) {
	config := BasicConfig{
		PluginType: "test-type",
	}
	context := plugin.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin config is missing the `id` field.")
}

func TestBasicConfigBuildWithoutType(t *testing.T) {
	config := BasicConfig{
		PluginID: "test-id",
	}
	context := plugin.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin config is missing the `type` field.")
}

func TestBasicConfigBuildMissingLogger(t *testing.T) {
	config := BasicConfig{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	context := plugin.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin build context is missing a logger.")
}

func TestBasicConfigBuildValid(t *testing.T) {
	config := BasicConfig{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	context := testutil.NewTestBuildContext(t)
	plugin, err := config.Build(context)
	require.NoError(t, err)
	require.Equal(t, "test-id", plugin.PluginID)
	require.Equal(t, "test-type", plugin.PluginType)
}

func TestBasicPluginID(t *testing.T) {
	plugin := BasicPlugin{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	require.Equal(t, "test-id", plugin.ID())
}

func TestBasicPluginType(t *testing.T) {
	plugin := BasicPlugin{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	require.Equal(t, "test-type", plugin.Type())
}

func TestBasicPluginLogger(t *testing.T) {
	logger := &zap.SugaredLogger{}
	plugin := BasicPlugin{
		PluginID:      "test-id",
		PluginType:    "test-type",
		SugaredLogger: logger,
	}
	require.Equal(t, logger, plugin.Logger())
}

func TestBasicPluginStart(t *testing.T) {
	plugin := BasicPlugin{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	err := plugin.Start()
	require.NoError(t, err)
}

func TestBasicPluginStop(t *testing.T) {
	plugin := BasicPlugin{
		PluginID:   "test-id",
		PluginType: "test-type",
	}
	err := plugin.Stop()
	require.NoError(t, err)
}
