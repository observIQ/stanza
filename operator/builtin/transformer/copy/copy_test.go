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
package copy

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
	op        *CopyOperatorConfig
	input     func() *entry.Entry
	output    func() *entry.Entry
}

// Test building and processing a CopyOperatorConfig
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
			"body_to_body",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewRecordField("key2")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"key2": "val",
				}
				return e
			},
		},
		{
			"nested_to_body",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested", "nestedkey")
				cfg.To = entry.NewRecordField("key2")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"key2": "nestedval",
				}
				return e
			},
		},
		{
			"body_to_nested",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewRecordField("nested", "key2")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
						"key2":      "val",
					},
				}
				return e
			},
		},
		{
			"body_to_attribute",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewLabelField("key2")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
				}
				e.Attributes = map[string]string{"key2": "val"}
				return e
			},
		},
		{
			"attribute_to_body",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewLabelField("key")
				cfg.To = entry.NewRecordField("key2")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"key": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key": "val",
					"nested": map[string]interface{}{
						"nestedkey": "nestedval",
					},
					"key2": "val",
				}
				e.Attributes = map[string]string{"key": "val"}
				return e
			},
		},
		{
			"attribute_to_resource",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewLabelField("key")
				cfg.To = entry.NewResourceField("key2")
				return cfg
			}(),
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"key": "val"}
				return e
			},
			func() *entry.Entry {
				e := newTestEntry()
				e.Attributes = map[string]string{"key": "val"}
				e.Resource = map[string]string{"key2": "val"}
				return e
			},
		},
		{
			"overwrite",
			false,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("key")
				cfg.To = entry.NewRecordField("nested")
				return cfg
			}(),
			newTestEntry,
			func() *entry.Entry {
				e := newTestEntry()
				e.Record = map[string]interface{}{
					"key":    "val",
					"nested": "val",
				}
				return e
			},
		},
		{
			"invalid_copy_obj_to_resource",
			true,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewResourceField("invalid")
				return cfg
			}(),
			newTestEntry,
			nil,
		},
		{
			"invalid_copy_obj_to_attributes",
			true,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewRecordField("nested")
				cfg.To = entry.NewLabelField("invalid")
				return cfg
			}(),
			newTestEntry,
			nil,
		},
		{
			"invalid_key",
			true,
			func() *CopyOperatorConfig {
				cfg := defaultCfg()
				cfg.From = entry.NewLabelField("nonexistentkey")
				cfg.To = entry.NewResourceField("key2")
				return cfg
			}(),
			newTestEntry,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run("BuildAndProcess/"+tc.name, func(t *testing.T) {
			cfg := tc.op
			cfg.OutputIDs = []string{"fake"}
			cfg.OnError = "drop"
			ops, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			op := ops[0]

			copy := op.(*CopyOperator)
			fake := testutil.NewFakeOutput(t)
			copy.SetOutputs([]operator.Operator{fake})
			val := tc.input()
			err = copy.Process(context.Background(), val)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				fake.ExpectEntry(t, tc.output())
			}
		})
	}
}

func defaultCfg() *CopyOperatorConfig {
	return NewCopyOperatorConfig("copy")
}
