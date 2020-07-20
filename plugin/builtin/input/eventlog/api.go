// +build windows

package eventlog

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	api           = windows.NewLazySystemDLL("wevtapi.dll")
	subscribeProc = api.NewProc("EvtSubscribe")
	nextProc      = api.NewProc("EvtNext")
	renderProc    = api.NewProc("EvtRender")
	closeProc     = api.NewProc("EvtClose")
)

const (
	// EvtSubscribeStartAtOldestRecord is a flag that will subscribe to all existing and future events.
	EvtSubscribeStartAtOldestRecord uint32 = 2
	// EvtSubscribeStartAfterBookmark is a flag that will subscribe to all events that begin after a bookmark.
	EvtSubscribeStartAfterBookmark uint32 = 3
)

// Subscribe will create a simplified pull subscription to a windows event channel.
// If a bookmark is supplied, the subscription will read all events after the bookmark.
// Otherwise, the subscription will begin at the oldest event.
func Subscribe(channelPath string, bookmark uintptr) (uintptr, error) {
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(signalEvent)
	signalEventPtr := uintptr(signalEvent)

	channelPtr, err := syscall.UTF16PtrFromString(channelPath)
	if err != nil {
		return 0, err
	}

	var flags uint32
	if bookmark == 0 {
		flags = EvtSubscribeStartAtOldestRecord
	} else {
		flags = EvtSubscribeStartAfterBookmark
	}

	return evtSubscribe(0, signalEventPtr, channelPtr, nil, bookmark, 0, 0, flags)
}

// GetEvents will return the next set of events on a subscription
func GetEvents(subscription uintptr, maxEvents int) ([]uintptr, error) {
	if maxEvents < 1 {
		maxEvents = 1
	}

	events := make([]uintptr, maxEvents)

	var eventsRead uint32
	if err := evtNext(subscription, uint32(maxEvents), &events[0], 0, 0, &eventsRead); err != nil {
		return nil, err
	}

	return events[:eventsRead], nil
}

// ReadEvents will read the events on a subscription
func ReadEvents(subscription uintptr, maxReads int) ([][]byte, error) {
	events, err := GetEvents(subscription, maxReads)
	if err != nil {
		return nil, err
	}

	defer func() {
		for _, event := range events {
			_ = evtClose(event)
		}
	}()

	results := [][]byte{}
	for _, event := range events {
		bytes, err := RenderEvent(event)
		if err != nil {
			return nil, err
		}

		results = append(results, bytes)
	}

	return results, nil
}

// RenderEvent will render an event as XML
func RenderEvent(event uintptr) ([]byte, error) {
	// TODO: Dynamic buffer size
	var buffer = make([]byte, 5000)
	var bufferUsed, propertyCount uint32
	if err := evtRender(0, event, 1, uint32(len(buffer)), &buffer[0], &bufferUsed, &propertyCount); err != nil {
		return []byte{}, nil
	}

	return buffer[:bufferUsed], nil
}

// evtSubscribe is the direct syscall implementation of EvtSubscribe (https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtsubscribe)
func evtSubscribe(session uintptr, signalEvent uintptr, channelPath *uint16, query *uint16, bookmark uintptr, context uintptr, callback uintptr, flags uint32) (uintptr, error) {
	handle, _, err := subscribeProc.Call(session, signalEvent, uintptr(unsafe.Pointer(channelPath)), uintptr(unsafe.Pointer(query)), bookmark, context, callback, uintptr(flags))
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
