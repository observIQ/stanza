// +build windows

package windows

import (
	"fmt"
)

// Event is an event stored in windows event log.
type Event struct {
	handle uintptr
}

// RenderSimple will render the event as EventXML without formatted info.
func (e *Event) RenderSimple(buffer Buffer) (EventXML, error) {
	if e.handle == 0 {
		return EventXML{}, fmt.Errorf("event handle does not exist")
	}

	var bufferUsed, propertyCount uint32
	err := evtRender(0, e.handle, EvtRenderEventXML, buffer.SizeBytes(), buffer.FirstByte(), &bufferUsed, &propertyCount)
	if err == ErrorInsufficientBuffer {
		buffer.UpdateSizeBytes(bufferUsed)
		return e.RenderSimple(buffer)
	}

	if err != nil {
		return EventXML{}, fmt.Errorf("syscall to 'EvtRender' failed: %s", err)
	}

	bytes, err := buffer.ReadBytes(bufferUsed)
	if err != nil {
		return EventXML{}, fmt.Errorf("failed to read bytes from buffer: %s", err)
	}

	return unmarshalEventXML(bytes)
}

// RenderFormatted will render the event as EventXML with formatted info.
func (e *Event) RenderFormatted(buffer Buffer, publisher Publisher) (EventXML, error) {
	if e.handle == 0 {
		return EventXML{}, fmt.Errorf("event handle does not exist")
	}

	var bufferUsed uint32
	err := evtFormatMessage(publisher.handle, e.handle, 0, 0, 0, EvtFormatMessageXML, buffer.SizeWide(), buffer.FirstByte(), &bufferUsed)
	if err == ErrorInsufficientBuffer {
		buffer.UpdateSizeWide(bufferUsed)
		return e.RenderFormatted(buffer, publisher)
	}

	if err != nil {
		return EventXML{}, fmt.Errorf("syscall to 'EvtFormatMessage' failed: %w, params: (%x, %x, 0, 0, 0, %x, %x, %p, %p)",
			err,
			publisher.handle, e.handle, EvtFormatMessageXML, buffer.SizeWide(), buffer.FirstByte(), &bufferUsed)
	}

	bytes, err := buffer.ReadWideChars(bufferUsed)
	if err != nil {
		return EventXML{}, fmt.Errorf("failed to read bytes from buffer: %s", err)
	}

	return unmarshalEventXML(bytes)
}

// Close will close the event handle.
func (e *Event) Close() error {
	if e.handle == 0 {
		return nil
	}

	if err := evtClose(e.handle); err != nil {
		return fmt.Errorf("failed to close event handle: %s", err)
	}

	e.handle = 0
	return nil
}

// NewEvent will create a new event from an event handle.
func NewEvent(handle uintptr) Event {
	return Event{
		handle: handle,
	}
}
