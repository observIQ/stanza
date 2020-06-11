package helper

import (
	"strconv"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/stretchr/testify/require"
)

func TestExprString(t *testing.T) {
	exampleEntry := func() *entry.Entry {
		e := entry.New()
		e.Record = map[string]interface{}{
			"test": "value",
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
			"{{ 'test' }}",
			"test",
		},
		{
			"prefix-{{ 'test' }}",
			"prefix-test",
		},
		{
			"prefix-{{ 'test' }}-suffix",
			"prefix-test-suffix",
		},
		{
			"prefix-{{ 'test' }}-suffix-{{ 'test2' + 'test3' }}",
			"prefix-test-suffix-test2test3",
		},
		{
			"{{ 'test' }}{{'asdf'}}",
			"testasdf",
		},
		{
			"{}",
			"{}",
		},
		{
			"{{}",
			"{{}",
		},
		{
			"}}{{",
			"}}{{",
		},
		{
			"my {{ $.test }}",
			"my value",
		},
		{
			"my {{$.test }}",
			"my value",
		},
		{
			"my {{$.test}}",
			"my value",
		},
	}

	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			exprString, err := tc.config.Build()
			require.NoError(t, err)

			env := map[string]interface{}{
				"$": exampleEntry().Record,
			}
			result, err := exprString.Render(env)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
