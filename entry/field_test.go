package entry

import (
	"encoding/json"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func testRecord() map[string]interface{} {
	return map[string]interface{}{
		"simple_key": "simple_value",
		"map_key":    nestedMap(),
	}
}

func nestedMap() map[string]interface{} {
	return map[string]interface{}{
		"nested_key": "nested_value",
	}
}

func TestFieldGet(t *testing.T) {
	cases := []struct {
		name        string
		field       Field
		record      map[string]interface{}
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			"EmptyField",
			[]string{},
			testRecord(),
			testRecord(),
			true,
		},
		{
			"NilField",
			nil,
			testRecord(),
			testRecord(),
			true,
		},
		{
			"SimpleKey",
			[]string{"simple_key"},
			testRecord(),
			"simple_value",
			true,
		},
		{
			"MapKey",
			[]string{"map_key"},
			testRecord(),
			nestedMap(),
			true,
		},
		{
			"NestedKey",
			[]string{"map_key", "nested_key"},
			testRecord(),
			"nested_value",
			true,
		},
		{
			"MissingKey",
			[]string{"invalid"},
			testRecord(),
			nil,
			false,
		},
		{
			"InvalidField",
			[]string{"simple_key", "nested_key"},
			testRecord(),
			nil,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Record = tc.record

			val, ok := entry.Get(tc.field)
			if !assert.Equal(t, tc.expectedOk, ok) {
				return
			}
			if !assert.Equal(t, tc.expectedVal, val) {
				return
			}
		})
	}
}

func TestFieldDelete(t *testing.T) {
	cases := []struct {
		name             string
		field            Field
		record           map[string]interface{}
		expectedRecord   interface{}
		expectedReturned interface{}
		expectedOk       bool
	}{
		{
			"SimpleKey",
			[]string{"simple_key"},
			testRecord(),
			map[string]interface{}{
				"map_key": nestedMap(),
			},
			"simple_value",
			true,
		},
		{
			"EmptyRecordAndField",
			[]string{},
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			true,
		},
		{
			"EmptyRecordAndNilField",
			nil,
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			true,
		},
		{
			"EmptyField",
			[]string{},
			testRecord(),
			map[string]interface{}{},
			testRecord(),
			true,
		},
		{
			"MissingKey",
			[]string{"missing_key"},
			testRecord(),
			testRecord(),
			nil,
			false,
		},
		{
			"NestedKey",
			[]string{"map_key", "nested_key"},
			testRecord(),
			map[string]interface{}{
				"simple_key": "simple_value",
				"map_key":    map[string]interface{}{},
			},
			"nested_value",
			true,
		},
		{
			"MapKey",
			[]string{"map_key"},
			testRecord(),
			map[string]interface{}{
				"simple_key": "simple_value",
			},
			nestedMap(),
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Record = tc.record

			deleted, ok := entry.Delete(tc.field)
			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedReturned, deleted)
			assert.Equal(t, tc.expectedRecord, entry.Record)
		})
	}
}

func TestFieldSet(t *testing.T) {
	cases := []struct {
		name        string
		selector    Field
		record      map[string]interface{}
		setTo       interface{}
		expectedVal interface{}
		expectedErr bool
	}{
		{
			"RootFieldRawValue",
			[]string{},
			testRecord(),
			"new_value",
			testRecord(),
			true,
		},
		{
			"NilFieldRawValue",
			nil,
			testRecord(),
			"new_value",
			testRecord(),
			true,
		},
		{
			"RootFieldMapValue",
			[]string{},
			map[string]interface{}{},
			testRecord(),
			testRecord(),
			false,
		},
		{
			"EmptyMap",
			[]string{"new_key"},
			map[string]interface{}{},
			"new_value",
			map[string]interface{}{"new_key": "new_value"},
			false,
		},
		{
			"NestedMap",
			[]string{"new_key", "nested_key"},
			map[string]interface{}{},
			"nested_value",
			map[string]interface{}{
				"new_key": map[string]interface{}{
					"nested_key": "nested_value",
				},
			},
			false,
		},
		{
			"OverwriteNestedMap",
			[]string{"map_key"},
			testRecord(),
			"new_value",
			map[string]interface{}{
				"simple_key": "simple_value",
				"map_key":    "new_value",
			},
			false,
		},
		{
			"MergedNestedValue",
			[]string{"map_key"},
			testRecord(),
			map[string]interface{}{
				"merged_key": "merged_value",
			},
			map[string]interface{}{
				"simple_key": "simple_value",
				"map_key": map[string]interface{}{
					"nested_key": "nested_value",
					"merged_key": "merged_value",
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Record = tc.record

			err := entry.Set(tc.selector, tc.setTo)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedVal, entry.Record)
		})
	}
}

func TestFieldDecode(t *testing.T) {
	type decodeTarget struct {
		Field    Field
		FieldPtr *Field
	}

	cases := []struct {
		name        string
		input       map[string]interface{}
		expected    decodeTarget
		expectedErr bool
	}{
		{
			"NilField",
			map[string]interface{}{"field": nil},
			decodeTarget{
				Field: Field(nil),
			},
			false,
		},
		{
			"EmptyField",
			map[string]interface{}{"field": ""},
			decodeTarget{
				Field: Field([]string{""}),
			},
			false,
		},
		{
			"RootField",
			map[string]interface{}{"field": "$"},
			decodeTarget{
				Field: Field(nil),
			},
			false,
		},
		{
			"SimpleField",
			map[string]interface{}{"field": "test"},
			decodeTarget{
				Field: Field([]string{"test"}),
			},
			false,
		},
		{
			"ComplexField",
			map[string]interface{}{"field": "$.test1.test2"},
			decodeTarget{
				Field: Field([]string{"test1", "test2"}),
			},
			false,
		},
		{
			"ComplexFieldWithRoot",
			map[string]interface{}{"field": "test1.test2"},
			decodeTarget{
				Field: Field([]string{"test1", "test2"}),
			},
			false,
		},
		{
			"SimpleFieldPointer",
			map[string]interface{}{"fieldPtr": "test"},
			decodeTarget{
				FieldPtr: func() *Field {
					var field = Field([]string{"test"})
					return &field
				}(),
			},
			false,
		},
		{
			"ComplexFieldPointer",
			map[string]interface{}{"fieldPtr": "test1.test2"},
			decodeTarget{
				FieldPtr: func() *Field {
					var field = Field([]string{"test1", "test2"})
					return &field
				}(),
			},
			false,
		},
		{
			"InvalidDecodeType",
			map[string]interface{}{"fieldPtr": []byte("test1")},
			decodeTarget{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var target decodeTarget
			cfg := &mapstructure.DecoderConfig{
				Result:     &target,
				DecodeHook: FieldDecoder,
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

func TestFieldUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			Field([]string{"test1"}),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			Field([]string{"test1", "test2"}),
		},
		{
			"RootField",
			[]byte(`"$"`),
			Field([]string{}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := json.Unmarshal(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldMarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    Field
		expected []byte
	}{
		{
			"SimpleField",
			Field([]string{"test1"}),
			[]byte(`"test1"`),
		},
		{
			"ComplexField",
			Field([]string{"test1", "test2"}),
			[]byte(`"test1.test2"`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := json.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, res)
		})
	}
}

func TestFieldUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			Field([]string{"test1"}),
		},
		{
			"UnquotedField",
			[]byte(`test1`),
			Field([]string{"test1"}),
		},
		{
			"RootField",
			[]byte(`"$"`),
			Field([]string{}),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			Field([]string{"test1", "test2"}),
		},
		{
			"ComplexFieldWithRoot",
			[]byte(`"$.test1.test2"`),
			Field([]string{"test1", "test2"}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := yaml.Unmarshal(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldMarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    interface{}
		expected []byte
	}{
		{
			"SimpleField",
			Field([]string{"test1"}),
			[]byte("test1\n"),
		},
		{
			"ComplexField",
			Field([]string{"test1", "test2"}),
			[]byte("test1.test2\n"),
		},
		{
			"EmptyField",
			Field([]string{}),
			[]byte("$\n"),
		},
		{
			"NilField",
			Field(nil),
			[]byte("$\n"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := yaml.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, res)
		})
	}
}
