package helper

import (
	"os"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/stretchr/testify/require"
)

func TestIdentifier(t *testing.T) {
	os.Setenv("TEST_METADATA_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_METADATA_PLUGIN_ENV")

	cases := []struct {
		name     string
		config   IdentifierConfig
		input    *entry.Entry
		expected *entry.Entry
	}{
		{
			"AddLabelLiteral",
			func() IdentifierConfig {
				cfg := NewIdentifierConfig()
				cfg.Resource = map[string]ExprStringConfig{
					"key1": "value1",
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Resource = map[string]string{
					"key1": "value1",
				}
				return e
			}(),
		},
		{
			"AddLabelExpr",
			func() IdentifierConfig {
				cfg := NewIdentifierConfig()
				cfg.Resource = map[string]ExprStringConfig{
					"key1": `EXPR("start" + "end")`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Resource = map[string]string{
					"key1": "startend",
				}
				return e
			}(),
		},
		{
			"AddLabelEnv",
			func() IdentifierConfig {
				cfg := NewIdentifierConfig()
				cfg.Resource = map[string]ExprStringConfig{
					"key1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Resource = map[string]string{
					"key1": "foo",
				}
				return e
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			identifier, err := tc.config.Build()
			require.NoError(t, err)

			err = identifier.Identify(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected.Resource, tc.input.Resource)
		})
	}
}
