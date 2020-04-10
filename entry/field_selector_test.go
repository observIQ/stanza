package entry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleFieldSelectorGet(t *testing.T) {
	standardRecord := map[string]interface{}{
		"testkey": "testval",
		"testnested": map[string]interface{}{
			"testnestedkey": "testnestedval",
		},
	}

	cases := []struct {
		name        string
		selector    SingleFieldSelector
		record      interface{}
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			"Empty Selector",
			[]string{},
			standardRecord,
			standardRecord,
			true,
		},
		{
			"String Field",
			[]string{"testkey"},
			standardRecord,
			"testval",
			true,
		},
		{
			"Map Field",
			[]string{"testnested"},
			standardRecord,
			standardRecord["testnested"],
			true,
		},
		{
			"Nested",
			[]string{"testnested", "testnestedkey"},
			standardRecord,
			"testnestedval",
			true,
		},
		{
			"Missing",
			[]string{"invalid"},
			standardRecord,
			nil,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := NewEntry()
			entry.Record = tc.record

			val, ok := entry.Get(tc.selector)
			if !assert.Equal(t, tc.expectedOk, ok) {
				return
			}
			if !assert.Equal(t, tc.expectedVal, val) {
				return
			}
		})
	}
}
