// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package retain

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
	expectErr bool
	op        *RetainOperatorConfig
	input     func() *entry.Entry
	output    func() *entry.Entry
}

// Test building and processing a given config
func TestBuildAndProcess(t *testing.T) {
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

	cases := []testCase{
		{
			"retain_single",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("key"))
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
				}
				return e
			},
		},
		{
			"retain_multi",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("key"))
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("nested2"))
				return cfg
			}(),
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
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested2": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"retain_nest",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("nested2"))
				return cfg
			}(),
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
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"nested2": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				return e
			},
		},
		{
			"retain_nested_value",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("nested2", "nestedkey2"))
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"nested2": map[string]interface{}{
						"nestedkey2": "nestedval",
					},
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"nested2": map[string]interface{}{
						"nestedkey2": "nestedval",
					},
				}
				return e
			},
		},
		{
			"retain_single_attribute",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewLabelField("key"))
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
				e.Attributes = map[string]string{
					"key": "val",
				}
				return e
			},
		},
		{
			"retain_multi_attribute",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewLabelField("key1"))
				cfg.Fields = append(cfg.Fields, entry.NewLabelField("key2"))
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{
					"key1": "val",
					"key2": "val",
					"key3": "val",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{
					"key1": "val",
					"key2": "val",
				}
				return e
			},
		},
		{
			"retain_single_resource",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewResourceField("key"))
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
				e.Resource = map[string]string{
					"key": "val",
				}
				return e
			},
		},
		{
			"retain_multi_resource",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewResourceField("key1"))
				cfg.Fields = append(cfg.Fields, entry.NewResourceField("key2"))
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{
					"key1": "val",
					"key2": "val",
					"key3": "val",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{
					"key1": "val",
					"key2": "val",
				}
				return e
			},
		},
		{
			"retain_one_of_each",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewResourceField("key1"))
				cfg.Fields = append(cfg.Fields, entry.NewLabelField("key3"))
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("key"))
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{
					"key1": "val",
					"key2": "val",
				}
				e.Attributes = map[string]string{
					"key3": "val",
					"key4": "val",
				}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Resource = map[string]string{
					"key1": "val",
				}
				e.Attributes = map[string]string{
					"key3": "val",
				}
				e.Record = map[string]interface{}{
					"key": "val",
				}
				return e
			},
		},
		{
			"retain_a_non_existent_key",
			false,
			func() *RetainOperatorConfig {
				cfg := defaultCfg()
				cfg.Fields = append(cfg.Fields, entry.NewRecordField("aNonExsistentKey"))
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = nil
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

			retain := op.(*RetainOperator)
			fake := testutil.NewFakeOutput(t)
			retain.SetOutputs([]operator.Operator{fake})
			val := tc.input()
			err = retain.Process(context.Background(), val)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				fake.ExpectEntry(t, tc.output())
			}
		})
	}
}

func defaultCfg() *RetainOperatorConfig {
	return NewRetainOperatorConfig("retain")
}
