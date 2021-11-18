package helper

import (
	"os"
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
)

func TestAttributer(t *testing.T) {
	os.Setenv("TEST_METADATA_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_METADATA_PLUGIN_ENV")

	cases := []struct {
		name     string
		config   AttributerConfig
		input    *entry.Entry
		expected *entry.Entry
	}{
		{
			"AddAttributeLiteral",
			func() AttributerConfig {
				cfg := NewAttributerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"attribute1": "value1",
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"attribute1": "value1",
				}
				return e
			}(),
		},
		{
			"AddAttributeExpr",
			func() AttributerConfig {
				cfg := NewAttributerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"attribute1": `EXPR("start" + "end")`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"attribute1": "startend",
				}
				return e
			}(),
		},
		{
			"AddAttributeEnv",
			func() AttributerConfig {
				cfg := NewAttributerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"attribute1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"attribute1": "foo",
				}
				return e
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attributeer, err := tc.config.Build()
			require.NoError(t, err)

			err = attributeer.Attribute(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected.Attributes, tc.input.Attributes)
		})
	}
}
