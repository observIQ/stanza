package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func NewFakeRestructurePlugin() (*RestructurePlugin, *testutil.Plugin) {
	mock := testutil.Plugin{}
	logger, _ := zap.NewProduction()
	return &RestructurePlugin{
		TransformerPlugin: helper.TransformerPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:      "test",
				PluginType:    "restructure",
				SugaredLogger: logger.Sugar(),
			},
			Output: &mock,
		},
	}, &mock
}

func TestRestructurePlugin(t *testing.T) {
	newTestEntry := func() *entry.Entry {
		return &entry.Entry{
			Timestamp: time.Unix(1586632809, 0),
			Record: map[string]interface{}{
				"key": "val",
				"nested": map[string]interface{}{
					"nestedkey": "nestedval",
				},
			},
		}
	}
	cases := []struct {
		name   string
		ops    []Op
		input  *entry.Entry
		output *entry.Entry
	}{
		{
			name:   "Nothing",
			input:  newTestEntry(),
			output: newTestEntry(),
		},
		{
			name: "Remove",
			ops: []Op{
				{
					&OpRemove{entry.NewField("nested")},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
				}
				return e
			}(),
		},
		{
			name: "Retain",
			ops: []Op{
				{
					&OpRetain{[]entry.Field{entry.NewField("key")}},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
				}
				return e
			}(),
		},
		{
			name: "Move",
			ops: []Op{
				{
					&OpMove{
						From: entry.NewField("key"),
						To:   entry.NewField("newkey"),
					},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"newkey": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			}(),
		},
		{
			name: "Flatten",
			ops: []Op{
				{
					&OpFlatten{
						Field: entry.NewField("nested"),
					},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key":       "val",
					"nestedkey": "nestedval",
				}
				return e
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			plugin, mockOutput := NewFakeRestructurePlugin()
			plugin.ops = tc.ops

			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				if !assert.Equal(t, tc.output, args[1].(*entry.Entry)) {
					t.FailNow()
				}
			}).Return(nil)

			err := plugin.Process(context.Background(), tc.input)
			require.NoError(t, err)
		})
	}
}
