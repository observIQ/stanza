package builtin

import (
	"testing"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

			mockOutput.On("Process", mock.Anything).Run(func(args mock.Arguments) {
				if !assert.Equal(t, tc.output, args[0].(*entry.Entry)) {
					t.FailNow()
				}
			}).Return(nil)

			err := plugin.Process(tc.input)
			require.NoError(t, err)
		})
	}
}

func TestRestructureOpDecodeHook(t *testing.T) {
	cases := []struct {
		name             string
		input            interface{}
		expected         Op
		errorRequirement require.ErrorAssertionFunc
	}{
		{
			name: "OpAdd with value from map[string]interface{}",
			input: map[string]interface{}{
				"add": map[string]interface{}{
					"field": "message",
					"value": "newvalue",
				},
			},
			expected: Op{
				&OpAdd{
					Field: entry.NewField("message"),
					Value: "newvalue",
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpAdd with value from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"add": map[interface{}]interface{}{
					"field": "message",
					"value": "newvalue",
				},
			},
			expected: Op{
				&OpAdd{
					Field: entry.NewField("message"),
					Value: "newvalue",
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpAdd with value_expr from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"add": map[interface{}]interface{}{
					"field":      "message",
					"value_expr": `"newvalue"`,
				},
			},
			expected: Op{
				&OpAdd{
					Field: entry.NewField("message"),
					ValueExpr: func() *vm.Program {
						prog, err := expr.Compile(`"newvalue"`)
						require.NoError(t, err)
						return prog
					}(),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpRemove from map[string]interface{}",
			input: map[string]interface{}{
				"remove": "message",
			},
			expected: Op{
				&OpRemove{
					Field: entry.NewField("message"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpRemove from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"remove": "message",
			},
			expected: Op{
				&OpRemove{
					Field: entry.NewField("message"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpRetain from map[string]interface{}",
			input: map[string]interface{}{
				"retain": []string{"message"},
			},
			expected: Op{
				&OpRetain{
					Fields: []entry.Field{entry.NewField("message")},
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpRetain from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"retain": []string{"message"},
			},
			expected: Op{
				&OpRetain{
					Fields: []entry.Field{entry.NewField("message")},
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpRetain from map[string]interface{}",
			input: map[string]interface{}{
				"retain": []string{"message"},
			},
			expected: Op{
				&OpRetain{
					Fields: []entry.Field{entry.NewField("message")},
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpMove from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"move": map[interface{}]interface{}{
					"from": "message",
					"to":   "message2",
				},
			},
			expected: Op{
				&OpMove{
					From: entry.NewField("message"),
					To:   entry.NewField("message2"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpMove from map[string]interface{}",
			input: map[string]interface{}{
				"move": map[string]interface{}{
					"from": "message",
					"to":   "message2",
				},
			},
			expected: Op{
				&OpMove{
					From: entry.NewField("message"),
					To:   entry.NewField("message2"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpFlatten from map[string]interface{}",
			input: map[string]interface{}{
				"flatten": "message",
			},
			expected: Op{
				&OpFlatten{
					Field: entry.NewField("message"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "OpFlatten from map[interface{}]interface{}",
			input: map[interface{}]interface{}{
				"flatten": "message",
			},
			expected: Op{
				&OpFlatten{
					Field: entry.NewField("message"),
				},
			},
			errorRequirement: require.NoError,
		},
		{
			name: "Error invalid Op",
			input: map[string]interface{}{
				"invalid": "test",
			},
			errorRequirement: require.Error,
		},
		{
			name:             "Error invalid type",
			input:            42,
			errorRequirement: require.Error,
		},
		{
			name: "OpAdd error value and value_expr defined",
			input: map[string]interface{}{
				"add": map[string]interface{}{
					"field":      "message",
					"value":      "asdf",
					"value_expr": `"asdf"`,
				},
			},
			errorRequirement: require.Error,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var op Op
			cfg := &mapstructure.DecoderConfig{
				Result:     &op,
				DecodeHook: OpDecoder,
			}
			decoder, err := mapstructure.NewDecoder(cfg)
			require.NoError(t, err)

			err = decoder.Decode(tc.input)
			tc.errorRequirement(t, err)

			require.Equal(t, tc.expected, op)
		})
	}
}
