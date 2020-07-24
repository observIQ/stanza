// +build windows

package windows

import (
	"fmt"
	"syscall"
)

// Publisher is a windows metadata publisher.
type Publisher struct {
	handle uintptr
}

// Open will open the publisher handle with the supplied provider.
func (p *Publisher) Open(provider string) error {
	if p.handle != 0 {
		return fmt.Errorf("publisher handle is already open")
	}

	utf16, err := syscall.UTF16PtrFromString(provider)
	if err != nil {
		return fmt.Errorf("failed to convert provider to utf16: %s", err)
	}

	handle, err := evtOpenPublisherMetadata(0, utf16, nil, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to open publisher handle: %s", err)
	}

	p.handle = handle
	return nil
}

// Close will close the publisher handle.
func (p *Publisher) Close() error {
	if p.handle == 0 {
		return nil
	}

	if err := evtClose(p.handle); err != nil {
		return fmt.Errorf("failed to close publisher: %s", err)
	}

	p.handle = 0
	return nil
}

// NewPublisher will create a new publisher with an empty handle.
func NewPublisher() Publisher {
	return Publisher{
		handle: 0,
	}
}
