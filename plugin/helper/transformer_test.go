package helper

import (
	"context"
	"fmt"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTransformerConfigMissingBase(t *testing.T) {
	cfg := NewTransformerConfig("test", "")
	cfg.OutputIDs = []string{"test-output"}
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestTransformerConfigMissingOutput(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}

func TestTransformerConfigValid(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	cfg.OutputIDs = []string{"test-output"}
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}

func TestTransformerOnErrorDefault(t *testing.T) {
	cfg := NewTransformerConfig("test-id", "test-type")
	transformer, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.Equal(t, SendOnError, transformer.OnError)
}

func TestTransformerOnErrorInvalid(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	cfg.OnError = "invalid"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin config has an invalid `on_error` field.")
}

func TestTransformerConfigSetNamespace(t *testing.T) {
	cfg := NewTransformerConfig("test-id", "test-type")
	cfg.OutputIDs = []string{"test-output"}
	cfg.SetNamespace("test-namespace")
	require.Equal(t, "test-namespace.test-id", cfg.PluginID)
	require.Equal(t, "test-namespace.test-output", cfg.OutputIDs[0])
}

func TestTransformerPluginCanProcess(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	transformer, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.True(t, transformer.CanProcess())
}

func TestTransformerDropOnError(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerPlugin{
		OnError: DropOnError,
		WriterPlugin: WriterPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
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
		OnError: SendOnError,
		WriterPlugin: WriterPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
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
		OnError: SendOnError,
		WriterPlugin: WriterPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
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
