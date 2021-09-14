package move

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
)

type processTestCase struct {
	name      string
	expectErr bool
	op        *MoveOperatorConfig
	input     func() *entry.Entry
	output    func() *entry.Entry
}

func TestMoveProcess(t *testing.T) {
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

	cases := []processTestCase{
		{
			"MoveRecordToRecord",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewRecordField("new")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"new": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"MoveRecordToLabel",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewLabelField("new")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				e.Attributes = map[string]string{"new": "val"}
				return e
			},
		},
		{
			"MoveLabelToRecord",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewLabelField("new")
				cfg.To = entry.NewRecordField("new")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"new": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"new": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				e.Attributes = map[string]string{}
				return e
			},
		},
		{
			"MoveLabelToResource",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewLabelField("new")
				cfg.To = entry.NewResourceField("new")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"new": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{"new": "val"}
				e.Attributes = map[string]string{}
				return e
			},
		},
		{
			"MoveResourceToLabel",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewResourceField("new")
				cfg.To = entry.NewLabelField("new")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{"new": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{}
				e.Attributes = map[string]string{"new": "val"}
				return e
			},
		},
		{
			"MoveNest",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewRecordField("NewNested")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"NewNested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"MoveFromNestedObj",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested", "nestedkey")
				cfg.To = entry.NewRecordField("unnestedkey")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key":         "val",
					"nested":      map[string]interface{}{},
					"unnestedkey": "nestedval",
				}
				return e
			},
		},
		{
			"MoveToNestedObj",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("newnestedkey")
				cfg.To = entry.NewRecordField("nested", "newnestedkey")

				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"newnestedkey": "nestedval",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey":    "nestedval",
						"newnestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"MoveDoubleNestedObj",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested", "nested2")
				cfg.To = entry.NewRecordField("nested2")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
						"nested2": map[string]interface{}{
							"nestedkey": "nestedval",
						},
					},
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"nested2": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"MoveNestToResource",
			true,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewResourceField("NewNested")
				return cfg
			}(),
			newTestEntry,
			nil,
		},
		{
			"MoveNestToLabel",
			true,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewLabelField("NewNested")

				return cfg
			}(),
			newTestEntry,
			nil,
		},
		{
			"ReplaceRecordObj",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("wrapper")
				cfg.To = entry.NewRecordField()
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"wrapper": map[string]interface{}{
						"key": "val",
						"nested": map[string]interface{}{
							"nestedkey": "nestedval",
						},
					},
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"ReplaceRecordString",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewRecordField()
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = "val"
				return e
			},
		},
		{
			"MergeObjToRecord",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewRecordField()
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key":       "val",
					"nestedkey": "nestedval",
				}
				return e
			},
		},
	}
	for _, tc := range cases {
		t.Run("BuildandProcess/"+tc.name, func(t *testing.T) {
			cfg := tc.op
			cfg.OutputIDs = []string{"fake"}
			cfg.OnError = "drop"
			ops, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			op := ops[0]

			move := op.(*MoveOperator)
			fake := testutil.NewFakeOutput(t)
			move.SetOutputs([]operator.Operator{fake})
			val := tc.input()
			err = move.Process(context.Background(), val)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				fake.ExpectEntry(t, tc.output())
			}
		})
	}
}

func defaultCfg() *MoveOperatorConfig {
	return NewMoveOperatorConfig("move")
}
