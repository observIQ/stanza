package transformer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
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
		e := entry.New()
		e.Timestamp = time.Unix(1586632809, 0)
		e.Record = map[string]interface{}{
			"key": "val",
			"nested": map[string]interface{}{
				"nestedkey": "nestedval",
			},
		}
		return e
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
			name: "AddValue",
			ops: []Op{
				{
					&OpAdd{
						Field: entry.NewField("new"),
						Value: "message",
					},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record.(map[string]interface{})["new"] = "message"
				return e
			}(),
		},
		{
			name: "AddValueExpr",
			ops: []Op{
				{
					&OpAdd{
						Field: entry.NewField("new"),
						program: func() *vm.Program {
							vm, err := expr.Compile(`$.key + "_suffix"`)
							require.NoError(t, err)
							return vm
						}(),
					},
				},
			},
			input: newTestEntry(),
			output: func() *entry.Entry {
				e := newTestEntry()
				e.Record.(map[string]interface{})["new"] = "val_suffix"
				return e
			}(),
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
				require.Equal(t, tc.output, args[1].(*entry.Entry))
			}).Return(nil)

			err := plugin.Process(context.Background(), tc.input)
			require.NoError(t, err)
		})
	}
}

func TestRestructureSerializeRoundtrip(t *testing.T) {
	cases := []struct {
		name string
		op   Op
	}{
		{
			name: "AddValue",
			op: Op{&OpAdd{
				Field: entry.NewField("new"),
				Value: "message",
			}},
		},
		{
			name: "AddValueExpr",
			op: Op{&OpAdd{
				Field: entry.NewField("new"),
				ValueExpr: func() *string {
					s := `$.key + "_suffix"`
					return &s
				}(),
				program: func() *vm.Program {
					vm, err := expr.Compile(`$.key + "_suffix"`)
					require.NoError(t, err)
					return vm
				}(),
			}},
		},
		{
			name: "Remove",
			op:   Op{&OpRemove{entry.NewField("nested")}},
		},
		{
			name: "Retain",
			op:   Op{&OpRetain{[]entry.Field{entry.NewField("key")}}},
		},
		{
			name: "Move",
			op: Op{&OpMove{
				From: entry.NewField("key"),
				To:   entry.NewField("newkey"),
			}},
		},
		{
			name: "Flatten",
			op: Op{&OpFlatten{
				Field: entry.NewField("nested"),
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tc.op)
			require.NoError(t, err)

			var jsonOp Op
			err = json.Unmarshal(jsonBytes, &jsonOp)
			require.NoError(t, err)

			require.Equal(t, tc.op, jsonOp)

			yamlBytes, err := yaml.Marshal(tc.op)
			require.NoError(t, err)

			var yamlOp Op
			err = yaml.UnmarshalStrict(yamlBytes, &yamlOp)
			require.NoError(t, err)

			require.Equal(t, tc.op, yamlOp)
		})
	}
}

func TestUnmarshalAll(t *testing.T) {
	configYAML := `
type: restructure
id: my_restructure
output: test_output
ops:
  - add:
      field: "message"
      value: "val"
  - add:
      field: "message_suffix"
      value_expr: "$.message + \"_suffix\""
  - remove: "message"
  - retain:
      - "message_retain"
  - flatten: "message_flatten"
  - move:
      from: "message1"
      to: "message2"
`

	configJSON := `
{
  "type": "restructure",
  "id": "my_restructure",
  "output": "test_output",
  "ops": [{
    "add": {
      "field": "message",
      "value": "val"
    }
  },{
    "add": {
      "field": "message_suffix",
      "value_expr": "$.message + \"_suffix\""
    }
  },{
    "remove": "message"
  },{
    "retain": [
      "message_retain"
    ]
  },{
    "flatten": "message_flatten"
  },{
    "move": {
      "from": "message1",
      "to": "message2"
    }
  }]
}`

	expected := plugin.Config(plugin.Config{
		Builder: &RestructurePluginConfig{
			TransformerConfig: helper.TransformerConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "my_restructure",
					PluginType: "restructure",
				},
				OutputID: "test_output",
			},
			Ops: []Op{
				Op{&OpAdd{
					Field: entry.NewField("message"),
					Value: "val",
				}},
				Op{&OpAdd{
					Field: entry.NewField("message_suffix"),
					ValueExpr: func() *string {
						s := `$.message + "_suffix"`
						return &s
					}(),
					program: func() *vm.Program {
						vm, err := expr.Compile(`$.message + "_suffix"`)
						require.NoError(t, err)
						return vm
					}(),
				}},
				Op{&OpRemove{
					Field: entry.NewField("message"),
				}},
				Op{&OpRetain{
					Fields: []entry.Field{
						entry.NewField("message_retain"),
					},
				}},
				Op{&OpFlatten{
					Field: entry.NewField("message_flatten"),
				}},
				Op{&OpMove{
					From: entry.NewField("message1"),
					To:   entry.NewField("message2"),
				}},
			},
		},
	})

	var unmarshalledYAML plugin.Config
	err := yaml.UnmarshalStrict([]byte(configYAML), &unmarshalledYAML)
	require.NoError(t, err)
	require.Equal(t, expected, unmarshalledYAML)

	var unmarshalledJSON plugin.Config
	err = json.Unmarshal([]byte(configJSON), &unmarshalledJSON)
	require.NoError(t, err)
	require.Equal(t, expected, unmarshalledJSON)
}

func TestOpType(t *testing.T) {
	cases := []struct {
		op           OpApplier
		expectedType string
	}{
		{
			&OpAdd{},
			"add",
		},
		{
			&OpRemove{},
			"remove",
		},
		{
			&OpRetain{},
			"retain",
		},
		{
			&OpMove{},
			"move",
		},
		{
			&OpFlatten{},
			"flatten",
		},
	}

	for _, tc := range cases {
		t.Run(tc.expectedType, func(t *testing.T) {
			require.Equal(t, tc.expectedType, tc.op.Type())
		})
	}

	t.Run("InvalidOpType", func(t *testing.T) {
		raw := "- unknown: test"
		var ops []Op
		err := yaml.UnmarshalStrict([]byte(raw), &ops)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown op type")
	})
}
