// +build windows

package windows

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	api                = windows.NewLazySystemDLL("wevtapi.dll")
	subscribeProc      = api.NewProc("EvtSubscribe")
	nextProc           = api.NewProc("EvtNext")
	renderProc         = api.NewProc("EvtRender")
	closeProc          = api.NewProc("EvtClose")
	createBookmarkProc = api.NewProc("EvtCreateBookmark")
	updateBookmarkProc = api.NewProc("EvtUpdateBookmark")
)

const (
	// EvtSubscribeToFutureEvents is a flat that will subscribe to only future events.
	EvtSubscribeToFutureEvents uint32 = 1
	// EvtSubscribeStartAtOldestRecord is a flag that will subscribe to all existing and future events.
	EvtSubscribeStartAtOldestRecord uint32 = 2
	// EvtSubscribeStartAfterBookmark is a flag that will subscribe to all events that begin after a bookmark.
	EvtSubscribeStartAfterBookmark uint32 = 3
)

const (
	// ErrorInsufficientBuffer is an error code that indicates the data area passed to a system call is too small
	ErrorInsufficientBuffer syscall.Errno = 122
	// ErrorNoMoreItems is an error code that indicates no more data is available.
	ErrorNoMoreItems syscall.Errno = 259
)

// evtSubscribe is the direct syscall implementation of EvtSubscribe (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtsubscribe)
func evtSubscribe(session uintptr, signalEvent windows.Handle, channelPath *uint16, query *uint16, bookmark uintptr, context uintptr, callback uintptr, flags uint32) (uintptr, error) {
	handle, _, err := subscribeProc.Call(session, uintptr(signalEvent), uintptr(unsafe.Pointer(channelPath)), uintptr(unsafe.Pointer(query)), bookmark, context, callback, uintptr(flags))
	if handle == 0 {
		return 0, err
	}

	return handle, nil
}

// evtNext is the direct syscall implementation of EvtNext (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtnext)
func evtNext(resultSet uintptr, eventsSize uint32, events *uintptr, timeout uint32, flags uint32, returned *uint32) error {
	result, _, err := nextProc.Call(resultSet, uintptr(eventsSize), uintptr(unsafe.Pointer(events)), uintptr(timeout), uintptr(flags), uintptr(unsafe.Pointer(returned)))
	if result == 0 {
		return err
	}

	return nil
}

// evtRender is the direct syscall implementation of EvtRender (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtrender)
func evtRender(context uintptr, fragment uintptr, flags uint32, bufferSize uint32, buffer *byte, bufferUsed *uint32, propertyCount *uint32) error {
	result, _, err := renderProc.Call(context, fragment, uintptr(flags), uintptr(bufferSize), uintptr(unsafe.Pointer(buffer)), uintptr(unsafe.Pointer(bufferUsed)), uintptr(unsafe.Pointer(propertyCount)))
	if result == 0 {
		return err
	}

	return nil
}

// evtClose is the direct syscall implementation of EvtClose (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtclose)
func evtClose(handle uintptr) error {
	result, _, err := closeProc.Call(handle)
	if result == 0 {
		return err
	}

	return nil
}

// evtCreateBookmark is the direct syscall implementation of EvtCreateBookmark (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtcreatebookmark)
func evtCreateBookmark(bookmarkXML *uint16) (uintptr, error) {
	handle, _, err := createBookmarkProc.Call(uintptr(unsafe.Pointer(bookmarkXML)))
	if handle == 0 {
		return 0, err
	}

	return handle, nil
}

// evtUpdateBookmark is the direct syscall implementation of EvtUpdateBookmark (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtcreatebookmark)
func evtUpdateBookmark(bookmark uintptr, event uintptr) error {
	result, _, err := updateBookmarkProc.Call(bookmark, event)
	if result == 0 {
		return err
	}

	return nil
}
