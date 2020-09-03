package entry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelFieldGet(t *testing.T) {
	cases := []struct {
		name       string
		labels     map[string]string
		field      Field
		expected   interface{}
		expectedOK bool
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewLabelField("test"),
			"val",
			true,
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewLabelField("nonexistent"),
			"",
			false,
		},
		{
			"NilMap",
			nil,
			NewLabelField("nonexistent"),
			"",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Labels = tc.labels
			val, ok := entry.Get(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestLabelFieldDelete(t *testing.T) {
	cases := []struct {
		name           string
		labels         map[string]string
		field          Field
		expected       interface{}
		expectedOK     bool
		expectedLabels map[string]string
	}{
		{
			"Simple",
			map[string]string{
				"test": "val",
			},
			NewLabelField("test"),
			"val",
			true,
			map[string]string{},
		},
		{
			"NonexistentKey",
			map[string]string{
				"test": "val",
			},
			NewLabelField("nonexistent"),
			"",
			false,
			map[string]string{
				"test": "val",
			},
		},
		{
			"NilMap",
			nil,
			NewLabelField("nonexistent"),
			"",
			false,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := New()
			entry.Labels = tc.labels
			val, ok := entry.Delete(tc.field)
			require.Equal(t, tc.expectedOK, ok)
			require.Equal(t, tc.expected, val)
		})
	}
}

func TestLabelFieldSet(t *testing.T) {
	cases := []struct {
		name        string
		labels      map[string]string
		field       Field
		val         interface{}
		expected    map[string]string
		expectedErr bool
	}{
		{
			"Simple",
			map[string]string{},
			NewLabelField("test"),
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
			NewLabelField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NilMap",
			nil,
			NewLabelField("test"),
			"val",
			map[string]string{
				"test": "val",
			},
			false,
		},
		{
			"NonString",
			map[string]string{},
			NewLabelField("test"),
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
			entry.Labels = tc.labels
			err := entry.Set(tc.field, tc.val)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}

			require.Equal(t, tc.expected, entry.Labels)
		})
	}
}

func TestLabelFieldString(t *testing.T) {
	cases := []struct {
		name     string
		field    LabelField
		expected string
	}{
		{
			"Simple",
			LabelField{"foo"},
			"$labels.foo",
		},
		{
			"Empty",
			LabelField{""},
			"$labels.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.field.String())
		})
	}
}
