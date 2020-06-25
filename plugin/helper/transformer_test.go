package helper

import (
	"context"
	"fmt"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTransformerConfigMissingBase(t *testing.T) {
	config := TransformerConfig{
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `id` field.")
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
	require.Contains(t, err.Error(), "missing required `output` field")
}

func TestTransformerConfigValid(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestTransformerOnErrorDefault(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	context := testutil.NewBuildContext(t)
	transformer, err := config.Build(context)
	require.NoError(t, err)
	require.Equal(t, SendOnError, transformer.OnError)
}

func TestTransformerOnErrorInvalid(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
		OnError: "invalid",
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin config has an invalid `on_error` field.")
}

func TestTransformerConfigSetNamespace(t *testing.T) {
	config := TransformerConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	config.SetNamespace("test-namespace")
	require.Equal(t, "test-namespace.test-id", config.PluginID)
	require.Equal(t, "test-namespace.test-output", config.OutputIDs[0])
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

func TestTransformerDropOnError(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		OnError: DropOnError,
		WriterPlugin: WriterPlugin{
			OutputPlugins: []plugin.Plugin{output},
			OutputIDs:     []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) (*entry.Entry, error) {
		return e, fmt.Errorf("Failure")
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.Error(t, err)
	output.AssertNotCalled(t, "Process", mock.Anything, mock.Anything)
}

func TestTransformerSendOnError(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		OnError: SendOnError,
		WriterPlugin: WriterPlugin{
			OutputPlugins: []plugin.Plugin{output},
			OutputIDs:     []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) (*entry.Entry, error) {
		return e, fmt.Errorf("Failure")
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.NoError(t, err)
	output.AssertCalled(t, "Process", mock.Anything, mock.Anything)
}

func TestTransformerProcessWithValid(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		OnError: SendOnError,
		WriterPlugin: WriterPlugin{
			OutputPlugins: []plugin.Plugin{output},
			OutputIDs:     []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) (*entry.Entry, error) {
		return e, nil
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.NoError(t, err)
	output.AssertCalled(t, "Process", mock.Anything, mock.Anything)
}
