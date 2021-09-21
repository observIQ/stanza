package metadata

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	os.Setenv("TEST_METADATA_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_METADATA_PLUGIN_ENV")

	cases := []struct {
		name      string
		configMod func(*MetadataOperatorConfig)
		input     *entry.Entry
		expected  *entry.Entry
	}{
		{
			"AddAttributeLiteral",
			func(cfg *MetadataOperatorConfig) {
				cfg.Attributes = map[string]helper.ExprStringConfig{
					"attribute1": "value1",
				}
			},
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
			func(cfg *MetadataOperatorConfig) {
				cfg.Attributes = map[string]helper.ExprStringConfig{
					"attribute1": `EXPR("start" + "end")`,
				}
			},
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
			func(cfg *MetadataOperatorConfig) {
				cfg.Attributes = map[string]helper.ExprStringConfig{
					"attribute1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
			},
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Attributes = map[string]string{
					"attribute1": "foo",
				}
				return e
			}(),
		},
		{
			"AddResourceLiteral",
			func(cfg *MetadataOperatorConfig) {
				cfg.Resource = map[string]helper.ExprStringConfig{
					"key1": "value1",
				}
			},
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
			"AddResourceExpr",
			func(cfg *MetadataOperatorConfig) {
				cfg.Resource = map[string]helper.ExprStringConfig{
					"key1": `EXPR("start" + "end")`,
				}
			},
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
			"AddResourceEnv",
			func(cfg *MetadataOperatorConfig) {
				cfg.Resource = map[string]helper.ExprStringConfig{
					"key1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
			},
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
			cfg := NewMetadataOperatorConfig("test_operator_id")
			cfg.OutputIDs = []string{"fake"}
			tc.configMod(cfg)
			ops, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			op := ops[0]

			fake := testutil.NewFakeOutput(t)
			err = op.SetOutputs([]operator.Operator{fake})
			require.NoError(t, err)

			err = op.Process(context.Background(), tc.input)
			require.NoError(t, err)

			select {
			case e := <-fake.Received:
				require.Equal(t, e.Attributes, tc.expected.Attributes)
				require.Equal(t, e.Resource, tc.expected.Resource)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}
