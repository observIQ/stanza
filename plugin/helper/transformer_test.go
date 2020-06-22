package helper

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestTransformerConfigMissingBase(t *testing.T) {
	config := TransformerConfig{
		OutputID: "test-output",
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin config is missing the `id` field.")
}

func TestTransformerConfigMissingOutput(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin config is missing the `output` field.")
}

func TestTransformerConfigValid(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		OutputID: "test-output",
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestTransformerConfigSetNamespace(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		OutputID: "test-output",
	}
	config.SetNamespace("test-namespace")
	require.Equal(t, "test-namespace.test-id", config.PluginID)
	require.Equal(t, "test-namespace.test-output", config.OutputID)
}

func TestTransformerPluginCanProcess(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.True(t, transformer.CanProcess())
}


func TestTransformerPluginCanOutput(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.True(t, transformer.CanOutput())
}

func TestTransformerPluginOutputs(t *testing.T) {
	output := &testutil.Plugin{}
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		Output: output,
	}
	require.Equal(t, []plugin.Plugin{output}, transformer.Outputs())
}

func TestTransformerPluginSetOutputsValid(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("CanProcess").Return(true)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		OutputID: "test-output",
	}

	err := transformer.SetOutputs([]plugin.Plugin{output})
	require.NoError(t, err)
	require.Equal(t, []plugin.Plugin{output}, transformer.Outputs())
}

func TestTransformerPluginSetOutputsInvalid(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("CanProcess").Return(false)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		OutputID: "test-output",
	}

	err := transformer.SetOutputs([]plugin.Plugin{output})
	require.Error(t, err)
}
