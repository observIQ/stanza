package helper

import (
	"os"
	"strconv"
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
)

func TestExprString(t *testing.T) {
	os.Setenv("TEST_EXPR_STRING_ENV", "foo")
	defer os.Unsetenv("TEST_EXPR_STRING_ENV")

	exampleEntry := func() *entry.Entry {
		e := entry.New()
		e.Body = map[string]interface{}{
			"test": "value",
		}
		e.Resource = map[string]string{
			"id": "value",
		}
		return e
	}

	cases := []struct {
		config   ExprStringConfig
		expected string
	}{
		{
			"test",
			"test",
		},
		{
			"EXPR( 'test' )",
			"test",
		},
		{
			"prefix-EXPR( 'test' )",
			"prefix-test",
		},
		{
			"prefix-EXPR('test')-suffix",
			"prefix-test-suffix",
		},
		{
			"prefix-EXPR('test')-suffix-EXPR('test2' + 'test3')",
			"prefix-test-suffix-test2test3",
		},
		{
			"EXPR('test' )EXPR('asdf')",
			"testasdf",
		},
		{
			"EXPR",
			"EXPR",
		},
		{
			"EXPR(",
			"EXPR(",
		},
		{
			")EXPR(",
			")EXPR(",
		},
		{
			"my EXPR( $.test )",
			"my value",
		},
		{
			"my EXPR($.test )",
			"my value",
		},
		{
			"my EXPR($.test)",
			"my value",
		},
		{
			"my EXPR(env('TEST_EXPR_STRING_ENV'))",
			"my foo",
		},
		{
			"EXPR( $resource.id )",
			"value",
		},
	}

	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			exprString, err := tc.config.Build()
			require.NoError(t, err)

			env := GetExprEnv(exampleEntry())
			defer PutExprEnv(env)

			result, err := exprString.Render(env)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
