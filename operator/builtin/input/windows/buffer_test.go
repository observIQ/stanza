// +build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/unicode"
)

func TestBufferReadBytes(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, b := range utf16 {
		buffer.buffer[i] = b
	}
	offset := uint32(len(utf16) / 2)
	bytes, err := buffer.ReadBytes(offset)
	require.NoError(t, err)
	require.Equal(t, utf8, bytes)
}

func TestBufferReadString(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, b := range utf16 {
		buffer.buffer[i] = b
	}
	offset := uint32(len(utf16) / 2)
	result, err := buffer.ReadString(offset)
	require.NoError(t, err)
	require.Equal(t, "test", result)
}

func TestBufferUpdateSize(t *testing.T) {
	buffer := NewBuffer()
	buffer.UpdateSize(1)
	require.Equal(t, 1*bytesPerWChar, len(buffer.buffer))
}

func TestBufferSize(t *testing.T) {
	buffer := NewBuffer()
	require.Equal(t, uint32(defaultBufferSize/bytesPerWChar), buffer.Size())
}

func TestBufferFirstByte(t *testing.T) {
	buffer := NewBuffer()
	buffer.buffer[0] = '1'
	require.Equal(t, &buffer.buffer[0], buffer.FirstByte())
}
