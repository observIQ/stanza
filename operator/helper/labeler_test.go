package helper

import (
	"os"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
)

func TestLabeler(t *testing.T) {
	os.Setenv("TEST_METADATA_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_METADATA_PLUGIN_ENV")

	cases := []struct {
		name     string
		config   LabelerConfig
		input    *entry.Entry
		expected *entry.Entry
	}{
		{
			"AddLabelLiteral",
			func() LabelerConfig {
				cfg := NewLabelerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"label1": "value1",
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"label1": "value1",
				}
				return e
			}(),
		},
		{
			"AddLabelExpr",
			func() LabelerConfig {
				cfg := NewLabelerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"label1": `EXPR("start" + "end")`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"label1": "startend",
				}
				return e
			}(),
		},
		{
			"AddLabelEnv",
			func() LabelerConfig {
				cfg := NewLabelerConfig()
				cfg.Attributes = map[string]ExprStringConfig{
					"label1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"label1": "foo",
				}
				return e
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			labeler, err := tc.config.Build()
			require.NoError(t, err)

			err = labeler.Label(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected.Attributes, tc.input.Attributes)
		})
	}
}
