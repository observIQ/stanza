package transformer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"github.com/observiq/carbon/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	os.Setenv("TEST_METADATA_PLUGIN_ENV", "foo")
	defer os.Unsetenv("TEST_METADATA_PLUGIN_ENV")

	baseConfig := func() *MetadataOperatorConfig {
		cfg := NewMetadataOperatorConfig("test_operator_id")
		cfg.OutputIDs = []string{"output1"}
		return cfg
	}

	cases := []struct {
		name     string
		config   *MetadataOperatorConfig
		input    *entry.Entry
		expected *entry.Entry
	}{
		{
			"AddLabelLiteral",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Labels = map[string]helper.ExprStringConfig{
					"label1": "value1",
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Labels = map[string]string{
					"label1": "value1",
				}
				return e
			}(),
		},
		{
			"AddLabelExpr",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Labels = map[string]helper.ExprStringConfig{
					"label1": `EXPR("start" + "end")`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Labels = map[string]string{
					"label1": "startend",
				}
				return e
			}(),
		},
		{
			"AddLabelEnv",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Labels = map[string]helper.ExprStringConfig{
					"label1": `EXPR(env("TEST_METADATA_PLUGIN_ENV"))`,
				}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Labels = map[string]string{
					"label1": "foo",
				}
				return e
			}(),
		},
		{
			"AddResourceLiteral",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Resource = map[string]helper.ExprStringConfig{
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
			"AddResourceExpr",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Resource = map[string]helper.ExprStringConfig{
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
			"AddResourceEnv",
			func() *MetadataOperatorConfig {
				cfg := baseConfig()
				cfg.Resource = map[string]helper.ExprStringConfig{
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
			metadataOperator, err := tc.config.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)

			mockOutput := testutil.NewMockOperator("output1")
			entryChan := make(chan *entry.Entry, 1)
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				entryChan <- args.Get(1).(*entry.Entry)
			}).Return(nil)

			err = metadataOperator.SetOutputs([]operator.Operator{mockOutput})
			require.NoError(t, err)

			err = metadataOperator.Process(context.Background(), tc.input)
			require.NoError(t, err)

			select {
			case e := <-entryChan:
				require.Equal(t, e.Labels, tc.expected.Labels)
				require.Equal(t, e.Resource, tc.expected.Resource)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}
