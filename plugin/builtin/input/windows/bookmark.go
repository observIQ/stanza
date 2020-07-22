// +build windows

package windows

import (
	"fmt"
	"syscall"

	"golang.org/x/text/encoding/unicode"
)

const bookmarkBufferSize = 16384

// Bookmark is a windows event bookmark.
type Bookmark struct {
	handle uintptr
	xml    string
	buffer []byte
}

// IsOpen indicates if the bookmark handle is open.
func (b *Bookmark) IsOpen() bool {
	return b.handle != 0
}

// IsDefined indicates if the bookmark has an XML definition.
func (b *Bookmark) IsDefined() bool {
	return b.xml == ""
}

// Open will open the bookmark handle using the bookmark's XML.
func (b *Bookmark) Open() error {
	if b.IsOpen() {
		return fmt.Errorf("bookmark handle is already open")
	}

	if !b.IsDefined() {
		return nil
	}

	utf16, err := syscall.UTF16PtrFromString(b.xml)
	if err != nil {
		return fmt.Errorf("failed to convert bookmark xml to utf16: %s", err)
	}

	handle, err := evtCreateBookmark(utf16)
	if err != nil {
		return fmt.Errorf("failed to create bookmark handle from xml: %s", err)
	}

	b.handle = handle
	return nil
}

// Update will update the bookmark based on the supplied event handle.
func (b *Bookmark) Update(eventHandle uintptr) error {
	if !b.IsOpen() {
		handle, err := evtCreateBookmark(nil)
		if err != nil {
			return fmt.Errorf("failed to create new bookmark handle: %s", err)
		}
		b.handle = handle
	}

	if err := evtUpdateBookmark(b.handle, eventHandle); err != nil {
		return fmt.Errorf("failed to update bookmark: %s", err)
	}

	xml, err := b.render()
	if err != nil {
		return err
	}

	b.xml = xml
	return nil
}

// Close will close the bookmark handle.
func (b *Bookmark) Close() error {
	if !b.IsOpen() {
		return nil
	}

	if err := evtClose(b.handle); err != nil {
		return fmt.Errorf("failed to close bookmark handle: %s", err)
	}

	b.handle = 0
	return nil
}

// Render will render the bookmark as XML.
func (b *Bookmark) render() (string, error) {
	if !b.IsOpen() {
		return "", fmt.Errorf("bookmark handle is not open")
	}

	var bufferUsed, propertyCount uint32
	err := evtRender(0, b.handle, 1, uint32(len(b.buffer)), &b.buffer[0], &bufferUsed, &propertyCount)
	if err == ErrorInsufficientBuffer {
		b.buffer = make([]byte, bufferUsed)
		return b.render()
	}

	if err != nil {
		return "", fmt.Errorf("syscall 'evtRender' failed: %s", err)
	}

	utf16 := b.buffer[:bufferUsed]
	bytes, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(utf16)
	if err != nil {
		return "", fmt.Errorf("failed to convert bookmark to utf8 string: %s", err)
	}

	return string(bytes), nil
}

// NewBookmark will create a new bookmark.
func NewBookmark(xml string) Bookmark {
	return Bookmark{
		handle: 0,
		xml:    xml,
		buffer: make([]byte, bookmarkBufferSize),
	}
}
