package helper

import (
	"context"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestInputConfigMissingBase(t *testing.T) {
	config := InputConfig{
		WriteTo: entry.Field{},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `id` field.")
}

func TestInputConfigMissingOutput(t *testing.T) {
	config := InputConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriteTo: entry.Field{},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `output` field")
}

func TestInputConfigValid(t *testing.T) {
	config := InputConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriteTo: entry.Field{},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestInputConfigSetNamespace(t *testing.T) {
	config := InputConfig{
		BasicConfig: BasicConfig{
			PluginID:   "test-id",
			PluginType: "test-type",
		},
		WriteTo: entry.Field{},
		WriterConfig: WriterConfig{
			OutputIDs: []string{"test-output"},
		},
	}
	config.SetNamespace("test-namespace")
	require.Equal(t, "test-namespace.test-id", config.PluginID)
	require.Equal(t, "test-namespace.test-output", config.OutputIDs[0])
}

func TestInputPluginCanProcess(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	input := InputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	require.False(t, input.CanProcess())
}

func TestInputPluginProcess(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	input := InputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
	}
	entry := entry.New()
	ctx := context.Background()
	err := input.Process(ctx, entry)
	require.Error(t, err)
	require.Equal(t, err.Error(), "Plugin can not process logs.")
}

func TestInputPluginNewEntry(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	writeTo := entry.NewRecordField("test-field")
	input := InputPlugin{
		BasicPlugin: BasicPlugin{
			PluginID:      "test-id",
			PluginType:    "test-type",
			SugaredLogger: buildContext.Logger,
		},
		WriteTo: writeTo,
	}

	entry := input.NewEntry("test")
	value, exists := entry.Get(writeTo)
	require.True(t, exists)
	require.Equal(t, "test", value)
}
