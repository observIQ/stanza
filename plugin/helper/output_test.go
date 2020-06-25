package helper

import (
	"testing"

	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/require"
)

func TestOutputConfigMissingBase(t *testing.T) {
	config := OutputConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `id` field.")
}

func TestOutputConfigBuildValid(t *testing.T) {
	config := OutputConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestOutputConfigNamespace(t *testing.T) {
	config := OutputConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
	}
	config.SetNamespace("test-namespace")
	require.Equal(t, "test-namespace.test-id", config.ID())
}

func TestOutputPluginCanProcess(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.True(t, output.CanProcess())
}

func TestOutputPluginCanOutput(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.False(t, output.CanOutput())
}

func TestOutputPluginOutputs(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.Equal(t, []plugin.Plugin{}, output.Outputs())
}

func TestOutputPluginSetOutputs(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}

	err := output.SetOutputs([]plugin.Plugin{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin can not output")
}
