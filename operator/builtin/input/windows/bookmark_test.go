// +build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/unicode"
)

func TestOpenPreexisting(t *testing.T) {
	bookmark := Bookmark{handle: 5}
	err := bookmark.Open("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "bookmark handle is already open")
}

func TestOpenInvalidUTF8(t *testing.T) {
	bookmark := NewBookmark()
	invalidUTF8 := "\u0000"
	err := bookmark.Open(invalidUTF8)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert bookmark xml to utf16")
}

func TestOpenSyscallFailure(t *testing.T) {
	bookmark := NewBookmark()
	xml := "<bookmark><\\bookmark>"
	createBookmarkProc = SimpleMockProc(0, 0, ErrorNotSupported)
	err := bookmark.Open(xml)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create bookmark handle from xml")
}

func TestOpenSuccess(t *testing.T) {
	bookmark := NewBookmark()
	xml := "<bookmark><\\bookmark>"
	createBookmarkProc = SimpleMockProc(5, 0, ErrorSuccess)
	err := bookmark.Open(xml)
	require.NoError(t, err)
	require.Equal(t, 5, bookmark.handle)
}

func TestUpdateFailureOnCreateSyscall(t *testing.T) {
	event := NewEvent(1)
	bookmark := NewBookmark()
	createBookmarkProc = SimpleMockProc(0, 0, ErrorNotSupported)
	err := bookmark.Update(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "syscall to `EvtCreateBookmark` failed")
}

func TestUpdateFailureOnUpdateSyscall(t *testing.T) {
	event := NewEvent(1)
	bookmark := NewBookmark()
	createBookmarkProc = SimpleMockProc(1, 0, ErrorSuccess)
	updateBookmarkProc = SimpleMockProc(0, 0, ErrorNotSupported)
	err := bookmark.Update(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "syscall to `EvtUpdateBookmark` failed")
}

func TestUpdateSuccess(t *testing.T) {
	event := NewEvent(1)
	bookmark := NewBookmark()
	createBookmarkProc = SimpleMockProc(5, 0, ErrorSuccess)
	updateBookmarkProc = SimpleMockProc(1, 0, ErrorSuccess)
	err := bookmark.Update(event)
	require.NoError(t, err)
	require.Equal(t, 5, bookmark.handle)
}

func TestCloseWhenAlreadyClosed(t *testing.T) {
	bookmark := NewBookmark()
	err := bookmark.Close()
	require.NoError(t, err)
}

func TestCloseSyscallFailure(t *testing.T) {
	bookmark := Bookmark{handle: 5}
	closeProc = SimpleMockProc(0, 0, ErrorNotSupported)
	err := bookmark.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close bookmark handle")
}

func TestCloseSuccess(t *testing.T) {
	bookmark := Bookmark{handle: 5}
	closeProc = SimpleMockProc(1, 0, ErrorSuccess)
	err := bookmark.Close()
	require.NoError(t, err)
	require.Equal(t, 0, bookmark.handle)
}

func TestRenderWhenClosed(t *testing.T) {
	bookmark := NewBookmark()
	buffer := NewBuffer()
	_, err := bookmark.Render(buffer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bookmark handle is not open")
}

func TestRenderInvalidSyscall(t *testing.T) {
	bookmark := Bookmark{handle: 5}
	buffer := NewBuffer()
	renderProc = SimpleMockProc(0, 0, ErrorNotSupported)
	_, err := bookmark.Render(buffer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "syscall to 'EvtRender' failed")
}

func TestRenderValidSyscall(t *testing.T) {
	bookmark := Bookmark{handle: 5}
	buffer := NewBuffer()
	utf8 := []byte("<xml>")
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(utf8)

	renderProc = MockProc{
		call: func(a ...uintptr) (uintptr, uintptr, error) {
			for i, byte := range utf16 {
				buffer.buffer[i] = byte
			}
			return 1, 0, ErrorSuccess
		},
	}

	xml, err := bookmark.Render(buffer)
	require.NoError(t, err)
	require.Equal(t, "<xml>", xml)
}
