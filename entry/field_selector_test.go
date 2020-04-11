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

func TestSingleFieldSelectorSetSafe(t *testing.T) {
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
		setTo       interface{}
		expectedVal interface{}
		expectedSet bool
	}{
		{
			"Empty Selector Empty Record",
			[]string{},
			nil,
			"inserted",
			"inserted",
			true,
		},
		{
			"Empty selector Nonempty record",
			[]string{},
			standardRecord,
			"inserted",
			standardRecord,
			false,
		},
		{
			"Empty Map",
			[]string{"insertedKey"},
			map[string]interface{}{},
			"insertedVal",
			map[string]interface{}{"insertedKey": "insertedVal"},
			true,
		},
		{
			"Nested Map",
			[]string{"testnested", "insertedKey"},
			standardRecord,
			"insertedVal",
			map[string]interface{}{
				"testkey": "testval",
				"testnested": map[string]interface{}{
					"testnestedkey": "testnestedval",
					"insertedKey":   "insertedVal",
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := NewEntry()
			entry.Record = tc.record

			ok := entry.SetSafe(tc.selector, tc.setTo)
			assert.Equal(t, tc.expectedSet, ok)
			assert.Equal(t, tc.expectedVal, entry.Record)
		})
	}
}

func TestSingleFieldSelectorSet(t *testing.T) {
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
		setTo       interface{}
		expectedVal interface{}
	}{
		{
			"Empty Selector Empty Record",
			[]string{},
			nil,
			"inserted",
			"inserted",
		},
		{
			"Empty selector Nonempty record",
			[]string{},
			standardRecord,
			"inserted",
			"inserted",
		},
		{
			"Empty Map",
			[]string{"insertedKey"},
			map[string]interface{}{},
			"insertedVal",
			map[string]interface{}{"insertedKey": "insertedVal"},
		},
		{
			"Nested Map",
			[]string{"testnested", "insertedKey"},
			standardRecord,
			"insertedVal",
			map[string]interface{}{
				"testkey": "testval",
				"testnested": map[string]interface{}{
					"testnestedkey": "testnestedval",
					"insertedKey":   "insertedVal",
				},
			},
		},
		{
			"Overwrite Nested Map",
			[]string{"testnested"},
			standardRecord,
			"insertedVal",
			map[string]interface{}{
				"testkey":    "testval",
				"testnested": "insertedVal",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := NewEntry()
			entry.Record = tc.record

			entry.Set(tc.selector, tc.setTo)
			assert.Equal(t, tc.expectedVal, entry.Record)
		})
	}
}
