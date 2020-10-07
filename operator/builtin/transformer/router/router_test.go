package router

import (
	"context"
	"os"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
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
		expectedLabels map[string]string
	}{
		{
			"DefaultRoute",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					helper.NewLabelerConfig(),
					"true",
					[]string{"output1"},
				},
			},
			map[string]int{"output1": 1},
			nil,
		},
		{
			"NoMatch",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					helper.NewLabelerConfig(),
					`false`,
					[]string{"output1"},
				},
			},
			map[string]int{},
			nil,
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
					helper.NewLabelerConfig(),
					`$.message == "non_match"`,
					[]string{"output1"},
				},
				{
					helper.NewLabelerConfig(),
					`$.message == "test_message"`,
					[]string{"output2"},
				},
			},
			map[string]int{"output2": 1},
			nil,
		},
		{
			"MatchWithLabel",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewLabelerConfig(),
					`$.message == "non_match"`,
					[]string{"output1"},
				},
				{
					helper.LabelerConfig{
						Labels: map[string]helper.ExprStringConfig{
							"label-key": "label-value",
						},
					},
					`$.message == "test_message"`,
					[]string{"output2"},
				},
			},
			map[string]int{"output2": 1},
			map[string]string{
				"label-key": "label-value",
			},
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
					helper.NewLabelerConfig(),
					`env("TEST_ROUTER_PLUGIN_ENV") == "foo"`,
					[]string{"output1"},
				},
				{
					helper.NewLabelerConfig(),
					`true`,
					[]string{"output2"},
				},
			},
			map[string]int{"output1": 1},
			nil,
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
			var labels map[string]string

			mock1 := testutil.NewMockOperator("$.output1")
			mock1.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output1"] = results["output1"] + 1
				if entry, ok := args[1].(*entry.Entry); ok {
					labels = entry.Labels
				}
			})
			mock2 := testutil.NewMockOperator("$.output2")
			mock2.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output2"] = results["output2"] + 1
				if entry, ok := args[1].(*entry.Entry); ok {
					labels = entry.Labels
				}
			})

			routerOperator := newOperator.(*RouterOperator)
			err = routerOperator.SetOutputs([]operator.Operator{mock1, mock2})
			require.NoError(t, err)

			err = routerOperator.Process(context.Background(), tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expectedCounts, results)
			require.Equal(t, tc.expectedLabels, labels)
		})
	}
}
