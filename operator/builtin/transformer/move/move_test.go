package move

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
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
		e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("key")
				cfg.To = entry.NewBodyField("new")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("key")
				cfg.To = entry.NewAttributeField("new")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewAttributeField("new")
				cfg.To = entry.NewBodyField("new")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"new": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewAttributeField("new")
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
				cfg.To = entry.NewAttributeField("new")
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
				cfg.From = entry.NewBodyField("nested")
				cfg.To = entry.NewBodyField("NewNested")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("nested", "nestedkey")
				cfg.To = entry.NewBodyField("unnestedkey")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("newnestedkey")
				cfg.To = entry.NewBodyField("nested", "newnestedkey")

				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("nested", "nested2")
				cfg.To = entry.NewBodyField("nested2")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("nested")
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
				cfg.From = entry.NewBodyField("nested")
				cfg.To = entry.NewAttributeField("NewNested")

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
				cfg.From = entry.NewBodyField("wrapper")
				cfg.To = entry.NewBodyField()
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
				e.Body = map[string]interface{}{
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
				cfg.From = entry.NewBodyField("key")
				cfg.To = entry.NewBodyField()
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = "val"
				return e
			},
		},
		{
			"MergeObjToRecord",
			false,
			func() *MoveOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewBodyField("nested")
				cfg.To = entry.NewBodyField()
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
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
