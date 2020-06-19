package builtin

import (
	"context"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	cfg := CopyPluginConfig{
		BasicConfig: helper.BasicConfig{
			PluginID:   "my_copy",
			PluginType: "copy",
		},
		OutputIDs: []string{"output1", "output2"},
	}

	buildContext := plugin.NewTestBuildContext(t)
	copyPlugin, err := cfg.Build(buildContext)
	require.NoError(t, err)

	results := map[string]int{}

	mock1 := mocks.NewMockPlugin("output1")
	mock1.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		results["output1"] = results["output1"] + 1
	})
	mock2 := mocks.NewMockPlugin("output2")
	mock2.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		results["output2"] = results["output2"] + 1
	})

	err = copyPlugin.SetOutputs([]plugin.Plugin{mock1, mock2})
	require.NoError(t, err)

	e := entry.New()
	err = copyPlugin.Process(context.Background(), e)
	require.NoError(t, err)

	expected := map[string]int{
		"output1": 1,
		"output2": 1,
	}

	require.Equal(t, expected, results)

}
