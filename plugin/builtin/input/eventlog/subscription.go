package eventlog

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// Subscription is a windows event subscription
type Subscription struct {
	handle   uintptr
	channel  string
	bookmark uintptr
	flags    uint32
}

// Open will open the subscription
func (s *Subscription) Open() error {
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to create signal handle: %s", err)
	}
	defer windows.CloseHandle(signalEvent)
	signal := uintptr(signalEvent)

	channel, err := syscall.UTF16PtrFromString(s.channel)
	if err != nil {
		return fmt.Errorf("failed to convert channel to uint16: %s", s.channel)
	}

	handle, err := evtSubscribe(0, signal, channel, nil, s.bookmark, 0, 0, s.flags)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s channel: %s", s.channel, err)
	}

	s.handle = handle
	return nil
}

// Close will close the subscription
func (s *Subscription) Close() error {
	if s.handle == 0 {
		return nil
	}

	if err := evtClose(s.handle); err != nil {
		return fmt.Errorf("failed to close subscription handle: %s", err)
	}

	s.handle = 0
	return nil
}

// NewSubscription will create a new subscription
func NewSubscription(channel string, bookmark uintptr, flags uint32) Subscription {
	return Subscription{
		handle:   0,
		channel:  channel,
		bookmark: bookmark,
		flags:    flags,
	}
}
