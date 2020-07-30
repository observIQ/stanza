// +build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/unicode"
)

func TestBufferReadBytesValid(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, byte := range utf16 {
		buffer.buffer[i] = byte
	}
	offset := uint32(len(utf16))
	bytes, err := buffer.ReadBytes(offset)
	require.NoError(t, err)
	require.Equal(t, utf8, bytes)
}

func TestBufferReadBytesInvalid(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, byte := range utf16 {
		buffer.buffer[i] = byte
	}
	offset := uint32(len(utf16))
	_, err := buffer.ReadBytes(offset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert buffer contents to utf8")
}

func TestBufferReadStringValid(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, byte := range utf16 {
		buffer.buffer[i] = byte
	}
	offset := uint32(len(utf16))
	result, err := buffer.ReadString(offset)
	require.NoError(t, err)
	require.Equal(t, "test", result)
}

func TestBufferReadStringInvalid(t *testing.T) {
	buffer := NewBuffer()
	utf8 := []byte("test")
	utf16, _ := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)
	for i, byte := range utf16 {
		buffer.buffer[i] = byte
	}
	offset := uint32(len(utf16))
	_, err := buffer.ReadString(offset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert buffer contents to utf8")
}

func TestBufferUpdateSize(t *testing.T) {
	buffer := NewBuffer()
	buffer.UpdateSize(1)
	require.Equal(t, 1, len(buffer.buffer))
}

func TestBufferSize(t *testing.T) {
	buffer := NewBuffer()
	require.Equal(t, uint32(defaultBufferSize), buffer.Size())
}

func TestBufferFirstByte(t *testing.T) {
	buffer := NewBuffer()
	buffer.buffer[0] = '1'
	require.Equal(t, &buffer.buffer[0], buffer.FirstByte())
}
