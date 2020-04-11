package entry

import (
	"testing"

	"github.com/mitchellh/mapstructure"
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

func TestFieldSelectorDecode(t *testing.T) {
	type decodeTarget struct {
		Fs    FieldSelector
		Fsptr *FieldSelector
		S     string
	}

	cases := []struct {
		name        string
		input       map[string]interface{}
		expected    decodeTarget
		expectedErr bool
	}{
		{
			"simple",
			map[string]interface{}{"fs": "test"},
			decodeTarget{
				Fs: SingleFieldSelector([]string{"test"}),
			},
			false,
		},
		{
			"multi",
			map[string]interface{}{"fs": []string{"test1", "test2"}},
			decodeTarget{
				Fs: SingleFieldSelector([]string{"test1", "test2"}),
			},
			false,
		},
		{
			"simple pointer",
			map[string]interface{}{"fsptr": "test"},
			decodeTarget{
				Fsptr: func() *FieldSelector {
					var fs FieldSelector = SingleFieldSelector([]string{"test"})
					return &fs
				}(),
			},
			false,
		},
		{
			"multi pointer",
			map[string]interface{}{"fsptr": []string{"test1", "test2"}},
			decodeTarget{
				Fsptr: func() *FieldSelector {
					var fs FieldSelector = SingleFieldSelector([]string{"test1", "test2"})
					return &fs
				}(),
			},
			false,
		},
		{
			"bad type",
			map[string]interface{}{"fsptr": []byte("test1")},
			decodeTarget{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var target decodeTarget
			cfg := &mapstructure.DecoderConfig{
				Result:     &target,
				DecodeHook: FieldSelectorDecoder,
			}

			decoder, err := mapstructure.NewDecoder(cfg)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = decoder.Decode(tc.input)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expected, target)
		})
	}
}
