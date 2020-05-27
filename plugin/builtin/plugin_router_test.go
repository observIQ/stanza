package builtin

import (
	"context"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRouterPlugin(t *testing.T) {
	basicConfig := func() *RouterPluginConfig {
		return &RouterPluginConfig{
			BasicPluginConfig: helper.BasicPluginConfig{
				PluginID:   "test_plugin_id",
				PluginType: "router",
			},
		}
	}

	cases := []struct {
		name           string
		input          *entry.Entry
		routes         []RouterPluginRouteConfig
		expectedCounts map[string]int
	}{
		{
			"DefaultRoute",
			entry.New(),
			[]RouterPluginRouteConfig{
				{
					"true",
					"output1",
				},
			},
			map[string]int{"output1": 1},
		},
		{
			"NoMatch",
			entry.New(),
			[]RouterPluginRouteConfig{
				{
					`false`,
					"output1",
				},
			},
			map[string]int{},
		},
		{
			"SimpleMatch",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]RouterPluginRouteConfig{
				{
					`$.message == "non_match"`,
					"output1",
				},
				{
					`$.message == "test_message"`,
					"output2",
				},
			},
			map[string]int{"output2": 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := basicConfig()
			cfg.Routes = tc.routes

			buildContext := testutil.NewTestBuildContext(t)
			newPlugin, err := cfg.Build(buildContext)
			require.NoError(t, err)

			results := map[string]int{}

			mock1 := &testutil.Plugin{}
			mock1.On("ID").Return("output1")
			mock1.On("CanProcess").Return(true)
			mock1.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output1"] = results["output1"] + 1
			})
			mock2 := &testutil.Plugin{}
			mock2.On("ID").Return("output2")
			mock2.On("CanProcess").Return(true)
			mock2.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output2"] = results["output2"] + 1
			})

			routerPlugin := newPlugin.(*RouterPlugin)
			err = routerPlugin.SetOutputs([]plugin.Plugin{mock1, mock2})
			require.NoError(t, err)

			err = routerPlugin.Process(context.Background(), tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expectedCounts, results)
		})
	}
}
