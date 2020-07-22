// +build windows

package windows

import (
	"encoding/xml"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
)

const subscriptionBufferSize = 16384

// Subscription is a subscription to a windows eventlog channel.
type Subscription struct {
	handle   uintptr
	bookmark Bookmark
	channel  string
	maxReads int
	startAt  string
	buffer   []byte
}

// IsOpen indicates if the subscription handle is open.
func (s *Subscription) IsOpen() bool {
	return s.handle != 0
}

// Open will open the subscription handle.
func (s *Subscription) Open() error {
	if s.IsOpen() {
		return fmt.Errorf("subscription handle is already open")
	}

	if err := s.bookmark.Open(); err != nil {
		return fmt.Errorf("failed to open bookmark: %s", err)
	}

	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to create signal handle: %s", err)
	}
	defer windows.CloseHandle(signalEvent)

	channel, err := syscall.UTF16PtrFromString(s.channel)
	if err != nil {
		return fmt.Errorf("failed to convert channel to utf16: %s", s.channel)
	}

	handle, err := evtSubscribe(0, signalEvent, channel, nil, s.bookmark.handle, 0, 0, s.getFlags())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s channel: %s", s.channel, err)
	}

	s.handle = handle
	return nil
}

// Close will close the subscription
func (s *Subscription) Close() error {
	if !s.IsOpen() {
		return nil
	}

	if err := evtClose(s.handle); err != nil {
		return fmt.Errorf("failed to close subscription handle: %s", err)
	}
	s.handle = 0

	if err := s.bookmark.Close(); err != nil {
		return fmt.Errorf("failed to close bookmark: %s", err)
	}

	return nil
}

// Read will read events from the subscription
func (s *Subscription) Read() ([]Event, error) {
	if !s.IsOpen() {
		return nil, fmt.Errorf("subscription handle is not open")
	}

	if s.maxReads < 1 {
		return nil, fmt.Errorf("max reads must be greater than 0")
	}

	eventHandles := make([]uintptr, s.maxReads)
	var eventsRead uint32
	err := evtNext(s.handle, uint32(s.maxReads), &eventHandles[0], 0, 0, &eventsRead)
	if err != nil && err != ErrorNoMoreItems {
		return nil, err
	}

	eventHandles = eventHandles[:eventsRead]
	defer s.closeAndBookmark(eventHandles)

	events, err := s.renderAll(eventHandles)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// BookmarkXML will return the xml representation of the current bookmark
func (s *Subscription) BookmarkXML() string {
	return s.bookmark.xml
}

// RenderAll will render a collection of event handles.
func (s *Subscription) renderAll(eventHandles []uintptr) ([]Event, error) {
	events := make([]Event, 0, len(eventHandles))
	for _, handle := range eventHandles {
		event, err := s.render(handle)
		if err != nil {
			return nil, fmt.Errorf("failed to render event: %s", err)
		}
		events = append(events, event)
	}
	return events, nil
}

// Render will render an event from its handle.
func (s *Subscription) render(eventHandle uintptr) (Event, error) {
	var bufferUsed, propertyCount uint32
	err := evtRender(0, eventHandle, 1, uint32(len(s.buffer)), &s.buffer[0], &bufferUsed, &propertyCount)
	if err == ErrorInsufficientBuffer {
		s.buffer = make([]byte, bufferUsed)
		return s.render(eventHandle)
	}

	if err != nil {
		return Event{}, fmt.Errorf("syscall 'evtRender' failed: %s", err)
	}

	utf16 := s.buffer[:bufferUsed]
	bytes, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(utf16)
	if err != nil {
		return Event{}, fmt.Errorf("failed to convert event to utf8: %s", err)
	}

	var event Event
	err = xml.Unmarshal(bytes, &event)
	if err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal xml as event: %s", err)
	}

	return event, nil
}

// closeAndBookmark will close a collection of event handles and update the subscription bookmark.
func (s *Subscription) closeAndBookmark(eventHandles []uintptr) {
	for i, handle := range eventHandles {
		if i == len(eventHandles)-1 {
			_ = s.bookmark.Update(handle)
		}
		_ = evtClose(handle)
	}
}

// getFlags will return the flags required to open the subscription.
func (s *Subscription) getFlags() uint32 {
	if s.bookmark.IsDefined() {
		return EvtSubscribeStartAfterBookmark
	}

	if s.startAt == "beginning" {
		return EvtSubscribeStartAtOldestRecord
	}

	return EvtSubscribeToFutureEvents
}

// NewSubscription will create a new subscription
func NewSubscription(channel string, bookmarkXML string, maxReads int, startAt string) Subscription {
	return Subscription{
		handle:   0,
		channel:  channel,
		bookmark: NewBookmark(bookmarkXML),
		maxReads: maxReads,
		startAt:  startAt,
		buffer:   make([]byte, subscriptionBufferSize),
	}
}
