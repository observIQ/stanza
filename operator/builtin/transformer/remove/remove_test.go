package remove

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
)

type testCase struct {
	name      string
	op        *RemoveOperatorConfig
	input     func() *entry.Entry
	output    func() *entry.Entry
	expectErr bool
}

func TestProcessAndBuild(t *testing.T) {
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

	cases := []testCase{
		{
			"remove_one",
			func() *RemoveOperatorConfig {
				cfg := defaultCfg()
				cfg.Field = entry.NewBodyField("key")
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
				return e
			},
			false,
		},
		{
			"remove_nestedkey",
			func() *RemoveOperatorConfig {
				cfg := defaultCfg()
				cfg.Field = entry.NewBodyField("nested", "nestedkey")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
					"key":    "val",
					"nested": map[string]interface{}{},
				}
				return e
			},
			false,
		},
		{
			"remove_obj",
			func() *RemoveOperatorConfig {
				cfg := defaultCfg()
				cfg.Field = entry.NewBodyField("nested")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Body = map[string]interface{}{
					"key": "val",
				}
				return e
			},
			false,
		},
		{
			"remove_single_attribute",
			func() *RemoveOperatorConfig {
				cfg := defaultCfg()
				cfg.Field = entry.NewAttributeField("key")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{
					"key": "val",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{}
				return e
			},
			false,
		},
		{
			"remove_single_resource",
			func() *RemoveOperatorConfig {
				cfg := defaultCfg()
				cfg.Field = entry.NewResourceField("key")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{
					"key": "val",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{}
				return e
			},
			false,
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

			remove := op.(*RemoveOperator)
			fake := testutil.NewFakeOutput(t)
			remove.SetOutputs([]operator.Operator{fake})
			val := tc.input()
			err = remove.Process(context.Background(), val)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				fake.ExpectEntry(t, tc.output())
			}
		})
	}
}

func defaultCfg() *RemoveOperatorConfig {
	return NewRemoveOperatorConfig("remove")
}
