package transformer

import (
	"context"
	"os"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"github.com/observiq/carbon/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilterOperator(t *testing.T) {
	os.Setenv("TEST_FILTER_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_FILTER_PLUGIN_ENV")

	basicConfig := func() *FilterOperatorConfig {
		return &FilterOperatorConfig{
			TransformerConfig: helper.NewTransformerConfig("test_operator_id", "filter"),
		}
	}

	cases := []struct {
		name       string
		input      *entry.Entry
		expression string
		filtered   bool
	}{
		{
			"RecordMatch",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			`$.message == "test_message"`,
			true,
		},
		{
			"NoMatchRecord",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "invalid",
				},
			},
			`$.message == "test_message"`,
			false,
		},
		{
			"MatchLabel",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
				Labels: map[string]string{
					"key": "value",
				},
			},
			`$labels.key == "value"`,
			true,
		},
		{
			"NoMatchLabel",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			`$labels.key == "value"`,
			false,
		},
		{
			"MatchEnv",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			`env("TEST_FILTER_PLUGIN_ENV") == "foo"`,
			true,
		},
		{
			"NoMatchEnv",
			&entry.Entry{
				Record: map[string]interface{}{
					"message": "test_message",
				},
			},
			`env("TEST_FILTER_PLUGIN_ENV") == "bar"`,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := basicConfig()
			cfg.Expression = tc.expression

			buildContext := testutil.NewBuildContext(t)
			testOperator, err := cfg.Build(buildContext)
			require.NoError(t, err)

			filtered := true
			mockOutput := testutil.NewMockOperator("output")
			mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				filtered = false
			})

			filterOperator, ok := testOperator.(*FilterOperator)
			require.True(t, ok)

			filterOperator.OutputOperators = []operator.Operator{mockOutput}
			err = filterOperator.Process(context.Background(), tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.filtered, filtered)
		})
	}
}
