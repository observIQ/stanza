package entry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRecordFieldGet(t *testing.T) {
	cases := []struct {
		name        string
		field       Field
		record      interface{}
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			"EmptyField",
			NewRecordField(),
			testRecord(),
			testRecord(),
			true,
		},
		{
			"SimpleField",
			NewRecordField("simple_key"),
			testRecord(),
			"simple_value",
			true,
		},
		{
			"MapField",
			NewRecordField("map_key"),
			testRecord(),
			nestedMap(),
			true,
		},
		{
			"NestedField",
			NewRecordField("map_key", "nested_key"),
			testRecord(),
			"nested_value",
			true,
		},
		{
			"MissingField",
			NewRecordField("invalid"),
			testRecord(),
			nil,
			false,
		},
		{
			"InvalidField",
			NewRecordField("simple_key", "nested_key"),
			testRecord(),
			nil,
			false,
		},
		{
			"RawField",
			NewRecordField(),
			"raw string",
			"raw string",
			true,
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

func TestRecordFieldDelete(t *testing.T) {
	cases := []struct {
		name             string
		field            Field
		record           interface{}
		expectedRecord   interface{}
		expectedReturned interface{}
		expectedOk       bool
	}{
		{
			"SimpleKey",
			NewRecordField("simple_key"),
			testRecord(),
			map[string]interface{}{
				"map_key": nestedMap(),
			},
			"simple_value",
			true,
		},
		{
			"EmptyRecordAndField",
			NewRecordField(),
			map[string]interface{}{},
			nil,
			map[string]interface{}{},
			true,
		},
		{
			"EmptyField",
			NewRecordField(),
			testRecord(),
			nil,
			testRecord(),
			true,
		},
		{
			"MissingKey",
			NewRecordField("missing_key"),
			testRecord(),
			testRecord(),
			nil,
			false,
		},
		{
			"NestedKey",
			NewRecordField("map_key", "nested_key"),
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
			NewRecordField("map_key"),
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

			entry.Delete(tc.field)
			assert.Equal(t, tc.expectedRecord, entry.Record)
		})
	}
}

func TestRecordFieldSet(t *testing.T) {
	cases := []struct {
		name        string
		field       Field
		record      interface{}
		setTo       interface{}
		expectedVal interface{}
	}{
		{
			"OverwriteMap",
			NewRecordField(),
			testRecord(),
			"new_value",
			"new_value",
		},
		{
			"OverwriteRaw",
			NewRecordField(),
			"raw_value",
			"new_value",
			"new_value",
		},
		{
			"NewMapValue",
			NewRecordField(),
			map[string]interface{}{},
			testRecord(),
			testRecord(),
		},
		{
			"NewRootField",
			NewRecordField("new_key"),
			map[string]interface{}{},
			"new_value",
			map[string]interface{}{"new_key": "new_value"},
		},
		{
			"NewNestedField",
			NewRecordField("new_key", "nested_key"),
			map[string]interface{}{},
			"nested_value",
			map[string]interface{}{
				"new_key": map[string]interface{}{
					"nested_key": "nested_value",
				},
			},
		},
		{
			"OverwriteNestedMap",
			NewRecordField("map_key"),
			testRecord(),
			"new_value",
			map[string]interface{}{
				"simple_key": "simple_value",
				"map_key":    "new_value",
			},
		},
		{
			"MergedNestedValue",
			NewRecordField("map_key"),
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
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Record = tc.record
			entry.Set(tc.field, tc.setTo)
			assert.Equal(t, tc.expectedVal, entry.Record)
		})
	}
}

func TestRecordFieldParent(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		field := RecordField{[]string{"child"}}
		require.Equal(t, RecordField{[]string{}}, field.Parent())
	})

	t.Run("Root", func(t *testing.T) {
		field := RecordField{[]string{}}
		require.Equal(t, RecordField{[]string{}}, field.Parent())
	})
}

func TestFieldChild(t *testing.T) {
	field := RecordField{[]string{"parent"}}
	require.Equal(t, RecordField{[]string{"parent", "child"}}, field.Child("child"))
}
