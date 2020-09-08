package buffer

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	t.Run("binaryRoundTrip", func(t *testing.T) {
		cases := [...]Metadata{
			0: {
				read:              []*readEntry{},
				unreadStartOffset: 0,
				unreadCount:       0,
				deadRangeStart:    0,
				deadRangeLength:   0,
			},
			1: {
				read:              []*readEntry{},
				unreadStartOffset: 0,
				unreadCount:       50,
				deadRangeStart:    0,
				deadRangeLength:   0,
			},
			2: {
				read: []*readEntry{
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
				read:              []*readEntry{},
				unreadStartOffset: 0,
				unreadCount:       50,
				deadRangeStart:    10,
				deadRangeLength:   100,
			},
		}

		for i, md := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var buf bytes.Buffer
				err := md.MarshalBinary(&buf)
				require.NoError(t, err)

				md2 := Metadata{}
				err = md2.UnmarshalBinary(&buf)
				require.NoError(t, err)

				require.Equal(t, md, md2)
			})
		}
	})
}
