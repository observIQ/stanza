package entry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributeFieldGet(t *testing.T) {
	cases := []struct {
		name       string
		attributes map[string]string
		field      Field
		expected   interface{}
		expectedOK bool
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewAttributeField("test"),
			"val",
			true,
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewAttributeField("nonexistent"),
			"",
			false,
		},
		{
			"NilMap",
			nil,
			NewAttributeField("nonexistent"),
			"",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Attributes = tc.attributes
			val, ok := entry.Get(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestAttributeFieldDelete(t *testing.T) {
	cases := []struct {
		name               string
		attributes         map[string]string
		field              Field
		expected           interface{}
		expectedOK         bool
		expectedAttributes map[string]string
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewAttributeField("test"),
			"val",
			true,
			map[string]string{},
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewAttributeField("nonexistent"),
			"",
			false,
			map[string]string{
				"test": "val",
			},
		},
		{
			"NilMap",
			nil,
			NewAttributeField("nonexistent"),
			"",
			false,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Attributes = tc.attributes
			val, ok := entry.Delete(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestAttributeFieldSet(t *testing.T) {
	cases := []struct {
		name        string
		attributes  map[string]string
		field       Field
		val         interface{}
		expected    map[string]string
		expectedErr bool
	}{
		{
			"Simple",
			map[string]string{},
			NewAttributeField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"Overwrite",
			map[string]string{
				"test": "original",
			},
			NewAttributeField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NilMap",
			nil,
			NewAttributeField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NonString",
			map[string]string{},
			NewAttributeField("test"),
			123,
			map[string]string{
				"test": "val",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Attributes = tc.attributes
			err := entry.Set(tc.field, tc.val)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}

			require.Equal(t, tc.expected, entry.Attributes)
		})
	}
}

func TestAttributeFieldString(t *testing.T) {
	cases := []struct {
		name     string
		field    AttributeField
		expected string
	}{
		{
			"Simple",
			AttributeField{"foo"},
			"$attributes.foo",
		},
		{
			"Empty",
			AttributeField{""},
			"$attributes.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.field.String())
		})
	}
}
