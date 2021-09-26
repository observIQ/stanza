package cache

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMemory(t *testing.T) {
	m := NewMemory(11)
	require.NotNil(t, m.cache)
	require.Equal(t, 11, m.maxSize)
}
func TestMemory(t *testing.T) {
	startTime := time.Now()
	time.Sleep(time.Nanosecond * 2)

	cases := []struct {
		name   string
		cache  *Memory
		input  map[string]interface{}
		expect *Memory
	}{
		{
			"basic",
			func() *Memory {
				return NewMemory(3)
			}(),
			map[string]interface{}{
				"key": "value",
				"map-value": map[string]string{
					"x":   "y",
					"dev": "stanza",
				},
			},
			&Memory{
				cache: map[string]item{
					"key": {
						data: "value",
					},
					"map-value": {
						data: map[string]string{
							"x":   "y",
							"dev": "stanza",
						},
					},
				},
				maxSize: 3,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.input {
				tc.cache.Add(key, value)
				out, ok := tc.cache.Get(key)
				require.True(t, ok, "expected to get value from cache immediately after adding it")
				require.Equal(t, value, out, "expected value to equal the value that was added to the cache")
				require.Greater(t, tc.cache.cache[key].timestamp.Nanosecond(), startTime.Nanosecond(), "expected cache entry to have a timestamp greater than zero")
			}

			require.Equal(t, tc.expect.maxSize, tc.cache.maxSize)
			require.Equal(t, len(tc.expect.cache), len(tc.cache.cache))

			for expectKey, expectItem := range tc.expect.cache {
				actual, ok := tc.cache.Get(expectKey)
				require.True(t, ok)
				require.Equal(t, expectItem.data, actual)
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	m := NewMemory(1)

	input := map[int]interface{}{
		1:  1,
		2:  2,
		3:  3,
		4:  4,
		5:  5,
		6:  6,
		7:  7,
		8:  8,
		9:  9,
		10: 10,
	}

	for i, value := range input {
		key := strconv.Itoa(i)

		m.Add(key, value)
		out, ok := m.Get(key)
		require.True(t, ok, "expected to get value from cache immediately after adding it")
		require.Equal(t, value, out, "expected value to equal the value that was added to the cache")

		// make sure previous cache item was removed
		if i > 0 {
			key := strconv.Itoa(i - 1)
			_, ok := m.Get(key)
			require.False(t, ok, "expected cache to have removed previous entry")
		}

		require.Len(t, m.cache, 1, "expected cache to contain one item")
	}
}
