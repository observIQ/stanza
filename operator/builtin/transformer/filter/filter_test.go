package filter

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilterOperator(t *testing.T) {
	os.Setenv("TEST_FILTER_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_FILTER_PLUGIN_ENV")

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
			cfg := NewFilterOperatorConfig("test")
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

func TestFilterDropRatio(t *testing.T) {
	cfg := NewFilterOperatorConfig("test")
	cfg.Expression = `$.message == "test_message"`
	cfg.DropRatio = 0.5
	buildContext := testutil.NewBuildContext(t)
	testOperator, err := cfg.Build(buildContext)
	require.NoError(t, err)

	processedEntries := 0
	mockOutput := testutil.NewMockOperator("output")
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		processedEntries++
	})

	filterOperator, ok := testOperator.(*FilterOperator)
	filterOperator.OutputOperators = []operator.Operator{mockOutput}
	require.True(t, ok)

	testEntry := &entry.Entry{
		Record: map[string]interface{}{
			"message": "test_message",
		},
	}

	for i := 1; i < 11; i++ {
		rand.Seed(1)
		err = filterOperator.Process(context.Background(), testEntry)
		require.NoError(t, err)
	}

	for i := 1; i < 11; i++ {
		rand.Seed(2)
		err = filterOperator.Process(context.Background(), testEntry)
		require.NoError(t, err)
	}

	require.Equal(t, 10, processedEntries)
}
