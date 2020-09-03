package entry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceFieldGet(t *testing.T) {
	cases := []struct {
		name       string
		resources  map[string]string
		field      Field
		expected   interface{}
		expectedOK bool
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewResourceField("test"),
			"val",
			true,
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewResourceField("nonexistent"),
			"",
			false,
		},
		{
			"NilMap",
			nil,
			NewResourceField("nonexistent"),
			"",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Resource = tc.resources
			val, ok := entry.Get(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestResourceFieldDelete(t *testing.T) {
	cases := []struct {
		name              string
		resources         map[string]string
		field             Field
		expected          interface{}
		expectedOK        bool
		expectedResources map[string]string
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewResourceField("test"),
			"val",
			true,
			map[string]string{},
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewResourceField("nonexistent"),
			"",
			false,
			map[string]string{
				"test": "val",
			},
		},
		{
			"NilMap",
			nil,
			NewResourceField("nonexistent"),
			"",
			false,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Resource = tc.resources
			val, ok := entry.Delete(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestResourceFieldSet(t *testing.T) {
	cases := []struct {
		name        string
		resources   map[string]string
		field       Field
		val         interface{}
		expected    map[string]string
		expectedErr bool
	}{
		{
			"Simple",
			map[string]string{},
			NewResourceField("test"),
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
			NewResourceField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NilMap",
			nil,
			NewResourceField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NonString",
			map[string]string{},
			NewResourceField("test"),
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
			entry.Resource = tc.resources
			err := entry.Set(tc.field, tc.val)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}

			require.Equal(t, tc.expected, entry.Resource)
		})
	}
}

func TestResourceFieldString(t *testing.T) {
	cases := []struct {
		name     string
		field    ResourceField
		expected string
	}{
		{
			"Simple",
			ResourceField{"foo"},
			"$resource.foo",
		},
		{
			"Empty",
			ResourceField{""},
			"$resource.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.field.String())
		})
	}
}
