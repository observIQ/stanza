package disk

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	t.Run("nextFlushedRange", func(t *testing.T) {
		cases := []struct {
			name    string
			flushed []bool
			start   int
			end     int
			ok      bool
		}{
			{"Empty", []bool{}, 0, 0, false},
			{"AllFlushed", []bool{true, true}, 0, 2, true},
			{"NoneFlushed", []bool{false, false}, 0, 0, false},
			{"StartWithFlushed", []bool{true, true, false}, 0, 2, true},
			{"EndWithFlushed", []bool{false, true, true}, 1, 3, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				m := Metadata{
					read: make([]*diskEntry, 0, len(tc.flushed)),
				}
				for i := 0; i < len(tc.flushed); i++ {
					m.read = append(m.read, &diskEntry{flushed: tc.flushed[i]})
				}
				start, end, ok := m.nextFlushedRange()
				require.Equal(t, tc.start, start)
				require.Equal(t, tc.end, end)
				require.Equal(t, tc.ok, ok)
			})
		}
	})

	t.Run("binaryRoundTrip", func(t *testing.T) {
		cases := [...]Metadata{
			0: {
				read:              []*diskEntry{},
				unreadStartOffset: 0,
				unreadCount:       0,
				deadRangeStart:    0,
				deadRangeLength:   0,
			},
			1: {
				read:              []*diskEntry{},
				unreadStartOffset: 0,
				unreadCount:       50,
				deadRangeStart:    0,
				deadRangeLength:   0,
			},
			2: {
				read: []*diskEntry{
					{
						flushed:     false,
						length:      10,
						startOffset: 0,
					},
				},
				unreadStartOffset: 10,
				unreadCount:       50,
				deadRangeStart:    0,
				deadRangeLength:   0,
			},
			3: {
				read:              []*diskEntry{},
				unreadStartOffset: 0,
				unreadCount:       50,
				deadRangeStart:    10,
				deadRangeLength:   100,
			},
		}

		for i, md := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var buf bytes.Buffer
				md.Write(&buf)

				md2 := Metadata{}
				err := md2.Read(&buf)
				require.NoError(t, err)

				require.Equal(t, md, md2)
			})
		}
	})
}
