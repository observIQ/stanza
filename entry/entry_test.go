package entry

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	testEntry := &Entry{
		Record: map[string]interface{}{
			"string_field": "string_val",
			"byte_field":   []byte(`test`),
			"map_string_interface_field": map[string]interface{}{
				"nested": "interface_val",
			},
			"map_string_interface_nonstring_field": map[string]interface{}{
				"nested": 111,
			},
			"map_string_string_field": map[string]string{
				"nested": "string_val",
			},
			"map_interface_interface_field": map[interface{}]interface{}{
				"nested": "interface_val",
			},
			"map_interface_interface_nonstring_key_field": map[interface{}]interface{}{
				100: "interface_val",
			},
			"map_interface_interface_nonstring_value_field": map[interface{}]interface{}{
				"nested": 100,
			},
		},
	}

	t.Run("field not exist error", func(t *testing.T) {
		var s string
		err := testEntry.Read(NewRecordField("nonexistant_field"), &s)
		require.Error(t, err)
	})

	t.Run("unsupported type error", func(t *testing.T) {
		var s **string
		err := testEntry.Read(NewRecordField("string_field"), &s)
		require.Error(t, err)
	})

	t.Run("string", func(t *testing.T) {
		var s string
		err := testEntry.Read(NewRecordField("string_field"), &s)
		require.NoError(t, err)
		require.Equal(t, "string_val", s)
	})

	t.Run("string error", func(t *testing.T) {
		var s string
		err := testEntry.Read(NewRecordField("map_string_interface_field"), &s)
		require.Error(t, err)
	})

	t.Run("map[string]interface{}", func(t *testing.T) {
		var m map[string]interface{}
		err := testEntry.Read(NewRecordField("map_string_interface_field"), &m)
		require.NoError(t, err)
		require.Equal(t, map[string]interface{}{"nested": "interface_val"}, m)
	})

	t.Run("map[string]interface{} error", func(t *testing.T) {
		var m map[string]interface{}
		err := testEntry.Read(NewRecordField("string_field"), &m)
		require.Error(t, err)
	})

	t.Run("map[string]string from map[string]interface{}", func(t *testing.T) {
		var m map[string]string
		err := testEntry.Read(NewRecordField("map_string_interface_field"), &m)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"nested": "interface_val"}, m)
	})

	t.Run("map[string]string from map[string]interface{} err", func(t *testing.T) {
		var m map[string]string
		err := testEntry.Read(NewRecordField("map_string_interface_nonstring_field"), &m)
		require.Error(t, err)
	})

	t.Run("map[string]string from map[interface{}]interface{}", func(t *testing.T) {
		var m map[string]string
		err := testEntry.Read(NewRecordField("map_interface_interface_field"), &m)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"nested": "interface_val"}, m)
	})

	t.Run("map[string]string from map[interface{}]interface{} nonstring key error", func(t *testing.T) {
		var m map[string]string
		err := testEntry.Read(NewRecordField("map_interface_interface_nonstring_key_field"), &m)
		require.Error(t, err)
	})

	t.Run("map[string]string from map[interface{}]interface{} nonstring value error", func(t *testing.T) {
		var m map[string]string
		err := testEntry.Read(NewRecordField("map_interface_interface_nonstring_value_field"), &m)
		require.Error(t, err)
	})

	t.Run("interface{} from any", func(t *testing.T) {
		var i interface{}
		err := testEntry.Read(NewRecordField("map_interface_interface_field"), &i)
		require.NoError(t, err)
		require.Equal(t, map[interface{}]interface{}{"nested": "interface_val"}, i)
	})

	t.Run("string from []byte", func(t *testing.T) {
		var i string
		err := testEntry.Read(NewRecordField("byte_field"), &i)
		require.NoError(t, err)
		require.Equal(t, "test", i)
	})
}

func TestCopy(t *testing.T) {
	entry := New()
	entry.Severity = Severity(0)
	entry.SeverityText = "ok"
	entry.Timestamp = time.Time{}
	entry.Record = "test"
	entry.Labels = map[string]string{"label": "value"}
	entry.Resource = map[string]string{"resource": "value"}
	entryCopy := entry.Copy()

	entry.Severity = Severity(1)
	entry.SeverityText = "1"
	entry.Timestamp = time.Now()
	entry.Record = "new"
	entry.Labels = map[string]string{"label": "new value"}
	entry.Resource = map[string]string{"resource": "new value"}

	require.Equal(t, time.Time{}, entryCopy.Timestamp)
	require.Equal(t, Severity(0), entryCopy.Severity)
	require.Equal(t, "ok", entryCopy.SeverityText)
	require.Equal(t, map[string]string{"label": "value"}, entryCopy.Labels)
	require.Equal(t, map[string]string{"resource": "value"}, entryCopy.Resource)
	require.Equal(t, "test", entryCopy.Record)
}

func TestFieldFromString(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		output        Field
		expectedError bool
	}{
		{
			"SimpleRecord",
			"test",
			Field{RecordField{[]string{"test"}}},
			false,
		},
		{
			"PrefixedRecord",
			"$.test",
			Field{RecordField{[]string{"test"}}},
			false,
		},
		{
			"FullPrefixedRecord",
			"$record.test",
			Field{RecordField{[]string{"test"}}},
			false,
		},
		{
			"SimpleLabel",
			"$labels.test",
			Field{LabelField{"test"}},
			false,
		},
		{
			"LabelsTooManyFields",
			"$labels.test.bar",
			Field{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := fieldFromString(tc.input)
			if tc.expectedError {
				require.Error(t, err)
				return
			}

			require.Equal(t, tc.output, f)
		})
	}
}

func TestAddLabel(t *testing.T) {
	entry := Entry{}
	entry.AddLabel("label", "value")
	expected := map[string]string{"label": "value"}
	require.Equal(t, expected, entry.Labels)
}

func TestAddResourceKey(t *testing.T) {
	entry := Entry{}
	entry.AddResourceKey("key", "value")
	expected := map[string]string{"key": "value"}
	require.Equal(t, expected, entry.Resource)
}

func TestReadToInterfaceMapWithMissingField(t *testing.T) {
	entry := Entry{}
	field := NewLabelField("label")
	dest := map[string]interface{}{}
	err := entry.readToInterfaceMap(field, &dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can not be read as a map[string]interface{}")
}

func TestReadToStringMapWithMissingField(t *testing.T) {
	entry := Entry{}
	field := NewLabelField("label")
	dest := map[string]string{}
	err := entry.readToStringMap(field, &dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can not be read as a map[string]string")
}

func TestReadToInterfaceMissingField(t *testing.T) {
	entry := Entry{}
	field := NewLabelField("label")
	var dest interface{}
	err := entry.readToInterface(field, &dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can not be read as a interface{}")
}

func TestDefaultTimestamp(t *testing.T) {
	os.Setenv(defaultTimestampEnv, "2019-10-12T07:20:50.52Z")
	now = getNow()
	defer func() { now = getNow() }()
	defer os.Unsetenv(defaultTimestampEnv)

	e := New()
	expected := time.Date(2019, 10, 12, 7, 20, 50, int(520*time.Millisecond), time.UTC)
	require.Equal(t, expected, e.Timestamp)
	require.True(t, e.Timestamp.Equal(expected))
}
