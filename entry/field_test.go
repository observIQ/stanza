package entry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestFieldUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			NewRecordField("test1"),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewRecordField("test1", "test2"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewRecordField([]string{}...),
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
			NewRecordField("test1"),
			[]byte(`"test1"`),
		},
		{
			"ComplexField",
			NewRecordField("test1", "test2"),
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
			NewRecordField("test1"),
		},
		{
			"UnquotedField",
			[]byte(`test1`),
			NewRecordField("test1"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewRecordField([]string{}...),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewRecordField("test1", "test2"),
		},
		{
			"ComplexFieldWithRoot",
			[]byte(`"$.test1.test2"`),
			NewRecordField("test1", "test2"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := yaml.UnmarshalStrict(tc.input, &f)
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
			NewRecordField("test1"),
			[]byte("test1\n"),
		},
		{
			"ComplexField",
			NewRecordField("test1", "test2"),
			[]byte("test1.test2\n"),
		},
		{
			"EmptyField",
			NewRecordField(),
			[]byte("$record\n"),
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
