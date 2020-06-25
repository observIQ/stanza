package transformer

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	baseConfig := func() *MetadataPluginConfig {
		return &MetadataPluginConfig{
			TransformerConfig: helper.TransformerConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "test_plugin_id",
					PluginType: "metadata",
				},
				OutputID: "output1",
			},
		}
	}

	cases := []struct {
		name     string
		config   *MetadataPluginConfig
		input    *entry.Entry
		expected *entry.Entry
	}{
		{
			"AddTagLiteral",
			func() *MetadataPluginConfig {
				cfg := baseConfig()
				cfg.Tags = []helper.ExprStringConfig{"tag1"}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Tags = []string{"tag1"}
				return e
			}(),
		},
		{
			"AddTagExpr",
			func() *MetadataPluginConfig {
				cfg := baseConfig()
				cfg.Tags = []helper.ExprStringConfig{`prefix-EXPR( 'test1' )`}
				return cfg
			}(),
			entry.New(),
			func() *entry.Entry {
				e := entry.New()
				e.Tags = []string{"prefix-test1"}
				return e
			}(),
		},
		{
			"AddLabelLiteral",
			func() *MetadataPluginConfig {
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
			func() *MetadataPluginConfig {
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			metadataPlugin, err := tc.config.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)

			mockOutput := testutil.NewMockPlugin("output1")
			entryChan := make(chan *entry.Entry, 1)
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				entryChan <- args.Get(1).(*entry.Entry)
			}).Return(nil)

			err = metadataPlugin.SetOutputs([]plugin.Plugin{mockOutput})
			require.NoError(t, err)

			err = metadataPlugin.Process(context.Background(), tc.input)
			require.NoError(t, err)

			select {
			case e := <-entryChan:
				require.Equal(t, e.Tags, tc.expected.Tags)
				require.Equal(t, e.Labels, tc.expected.Labels)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}
