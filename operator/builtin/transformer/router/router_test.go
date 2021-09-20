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
		name               string
		input              *entry.Entry
		routes             []*RouterOperatorRouteConfig
		defaultOutput      helper.OutputIDs
		expectedCounts     map[string]int
		expectedAttributes map[string]string
	}{
		{
			"DefaultRoute",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					"true",
					[]string{"output1"},
				},
			},
			nil,
			map[string]int{"output1": 1},
			nil,
		},
		{
			"NoMatch",
			entry.New(),
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`false`,
					[]string{"output1"},
				},
			},
			nil,
			map[string]int{},
			nil,
		},
		{
			"SimpleMatch",
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`$.message == "non_match"`,
					[]string{"output1"},
				},
				{
					helper.NewAttributerConfig(),
					`$.message == "test_message"`,
					[]string{"output2"},
				},
			},
			nil,
			map[string]int{"output2": 1},
			nil,
		},
		{
			"MatchWithAttribute",
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`$.message == "non_match"`,
					[]string{"output1"},
				},
				{
					helper.AttributerConfig{
						Attributes: map[string]helper.ExprStringConfig{
							"attribute-key": "attribute-value",
						},
					},
					`$.message == "test_message"`,
					[]string{"output2"},
				},
			},
			nil,
			map[string]int{"output2": 1},
			map[string]string{
				"attribute-key": "attribute-value",
			},
		},
		{
			"MatchEnv",
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`env("TEST_ROUTER_PLUGIN_ENV") == "foo"`,
					[]string{"output1"},
				},
				{
					helper.NewAttributerConfig(),
					`true`,
					[]string{"output2"},
				},
			},
			nil,
			map[string]int{"output1": 1},
			nil,
		},
		{
			"UseDefault",
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`false`,
					[]string{"output1"},
				},
			},
			[]string{"output2"},
			map[string]int{"output2": 1},
			nil,
		},
		{
			"MatchBeforeDefault",
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "test_message",
				},
			},
			[]*RouterOperatorRouteConfig{
				{
					helper.NewAttributerConfig(),
					`true`,
					[]string{"output1"},
				},
			},
			[]string{"output2"},
			map[string]int{"output1": 1},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := basicConfig()
			cfg.Routes = tc.routes
			cfg.Default = tc.defaultOutput

			buildContext := testutil.NewBuildContext(t)
			ops, err := cfg.Build(buildContext)
			require.NoError(t, err)
			op := ops[0]

			results := map[string]int{}
			var attributes map[string]string

			mock1 := testutil.NewMockOperator("$.output1")
			mock1.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output1"] = results["output1"] + 1
				if entry, ok := args[1].(*entry.Entry); ok {
					attributes = entry.Attributes
				}
			})

			mock2 := testutil.NewMockOperator("$.output2")
			mock2.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				results["output2"] = results["output2"] + 1
				if entry, ok := args[1].(*entry.Entry); ok {
					attributes = entry.Attributes
				}
			})

			routerOperator := op.(*RouterOperator)
			err = routerOperator.SetOutputs([]operator.Operator{mock1, mock2})
			require.NoError(t, err)

			err = routerOperator.Process(context.Background(), tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expectedCounts, results)
			require.Equal(t, tc.expectedAttributes, attributes)
		})
	}
}
