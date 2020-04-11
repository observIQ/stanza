package builtin

import (
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func NewFakeRestructurePlugin() (*RestructurePlugin, *testutil.Plugin) {
	mock := testutil.Plugin{}
	logger, _ := zap.NewProduction()
	return &RestructurePlugin{
		BasicPlugin: helper.BasicPlugin{
			PluginID:      "test",
			PluginType:    "restructure",
			SugaredLogger: logger.Sugar(),
		},
		BasicTransformer: helper.BasicTransformer{
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
		move   []MoveConfig
		remove []entry.FieldSelector
		retain []entry.FieldSelector
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
			remove: []entry.FieldSelector{
				entry.NewSingleFieldSelector("nested"),
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
			retain: []entry.FieldSelector{
				entry.NewSingleFieldSelector("key"),
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
			move: []MoveConfig{
				{
					From: entry.NewSingleFieldSelector("key"),
					To:   entry.NewSingleFieldSelector("newkey"),
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			plugin, mockOutput := NewFakeRestructurePlugin()
			plugin.move = tc.move
			plugin.remove = tc.remove
			plugin.retain = tc.retain

			mockOutput.On("Process", mock.Anything).Run(func(args mock.Arguments) {
				if !assert.Equal(t, tc.output, args[0].(*entry.Entry)) {
					t.FailNow()
				}
			}).Return(nil)

			err := plugin.Process(tc.input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		})
	}
}
