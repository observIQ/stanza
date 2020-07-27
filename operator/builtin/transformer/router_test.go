package transformer

import (
	"context"
	"os"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRouterOperator(t *testing.T) {
	os.Setenv("TEST_ROUTER_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_ROUTER_PLUGIN_ENV")

	basicConfig := func() *RouterOperatorConfig {
		return &RouterOperatorConfig{
			BasicConfig: helper.BasicConfig{
				OperatorID:   "test_operator_id",
				OperatorType: "router",
			},
		}
	}

	cases := []struct {
		name           string
		input          *entry.Entry
		routes         []*RouterOperatorRouteConfig
		expectedCounts map[string]int
	}{
		{
			"DefaultRoute",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					"true",
					[]string{"output1"},
				},
			},
			map[string]int{"output1": 1},
		},
		{
			"NoMatch",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					`false`,
					[]string{"output1"},
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
			[]*RouterOperatorRouteConfig{
				{
					`$.message == "non_match"`,
					[]string{"output1"},
				},
				{
					`$.message == "test_message"`,
					[]string{"output2"},
				},
			},
			map[string]int{"output2": 1},
		},
		{
			"MatchEnv",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					`env("TEST_ROUTER_PLUGIN_ENV") == "foo"`,
					[]string{"output1"},
				},
				{
					`true`,
					[]string{"output2"},
				},
			},
			map[string]int{"output1": 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := basicConfig()
			cfg.Routes = tc.routes

			buildContext := testutil.NewBuildContext(t)
			newOperator, err := cfg.Build(buildContext)
			require.NoError(t, err)

			results := map[string]int{}

			mock1 := testutil.NewMockOperator("output1")
			mock1.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output1"] = results["output1"] + 1
			})
			mock2 := testutil.NewMockOperator("output2")
			mock2.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output2"] = results["output2"] + 1
			})

			routerOperator := newOperator.(*RouterOperator)
			err = routerOperator.SetOutputs([]operator.Operator{mock1, mock2})
			require.NoError(t, err)

			err = routerOperator.Process(context.Background(), tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expectedCounts, results)
		})
	}
}
