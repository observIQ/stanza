package buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

var _ Buffer = (*MemoryBuffer)(nil)

// MemoryBufferConfig holds the configuration for a memory buffer
type MemoryBufferConfig struct {
	Type          string          `json:"type"        yaml:"type"`
	MaxEntries    int             `json:"max_entries" yaml:"max_entries"`
	MaxChunkDelay helper.Duration `json:"max_delay"   yaml:"max_delay"`
	MaxChunkSize  uint            `json:"max_chunk_size" yaml:"max_chunk_size"`
}

// NewMemoryBufferConfig creates a new default MemoryBufferConfig
func NewMemoryBufferConfig() *MemoryBufferConfig {
	return &MemoryBufferConfig{
		Type:          "memory",
		MaxEntries:    1 << 20,
		MaxChunkDelay: helper.NewDuration(time.Second),
		MaxChunkSize:  1000,
	}
}

// Build builds a MemoryBufferConfig into a Buffer, loading any entries that were previously unflushed
// back into memory
func (c MemoryBufferConfig) Build(operatorID string) (Buffer, error) {
	return &MemoryBuffer{
		operatorID:    operatorID,
		buf:           make(chan *entry.Entry, c.MaxEntries),
		maxChunkDelay: c.MaxChunkDelay.Raw(),
		maxChunkSize:  c.MaxChunkSize,
		closed:        false,
	}, nil
}

// MemoryBuffer is a buffer that holds all entries in memory until Close() is called.
// Once close is called all entries will be lost it is reccommended to call Drain before Close.
type MemoryBuffer struct {
	operatorID    string
	buf           chan *entry.Entry
	maxChunkDelay time.Duration
	maxChunkSize  uint
	closed        bool

	// readLock ensures only one concurrent read/drain can process at a time.
	// If multiple reads or a drain happen while another is processing it may cause
	// an indefinite block or longer blocks to maxChunkDelay.
	readLock sync.Mutex
}

// Add adds an entry onto the buffer.
// Is a blocking call if the buffer is full
func (m *MemoryBuffer) Add(ctx context.Context, e *entry.Entry) error {
	// If buffer is closed don't allow this operation
	if m.closed {
		return ErrBufferedClosed
	}

	// Insert into buffer or error if context finishes before we can
	select {
	case <-ctx.Done():
		return fmt.Errorf("ctx error adding to buffer: %w", ctx.Err())
	case m.buf <- e:
		return nil
	}
}

// Read reads from the buffer.
// Read will block until the there are MaxChunkSize entries or we have block as long as MachChunkDelay.
func (m *MemoryBuffer) Read(ctx context.Context) ([]*entry.Entry, error) {
	// If buffer is closed don't allow this operation
	if m.closed {
		return nil, ErrBufferedClosed
	}

	m.readLock.Lock()
	defer m.readLock.Unlock()

	entries := make([]*entry.Entry, 0, m.maxChunkSize)

	// Time for max Chunk delay
	// Using a timer here rather than a context with deadline as we don't want to confuse a context timeout with a maxChunkDelay timeout
	timer := time.NewTimer(m.maxChunkDelay)
	defer timer.Stop()

LOOP:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ctx error during buffer read: %w", ctx.Err())
		case <-timer.C:
			// Timer has reached maxChunkDelay break and return
			break LOOP
		case e := <-m.buf:
			entries = append(entries, e)

			// If we've reached the maxChunkSize break and return
			if len(entries) == int(m.maxChunkSize) {
				break LOOP
			}
		}
	}

	return entries, nil
}

// Close runs cleanup code for buffer
func (m *MemoryBuffer) Close() ([]*entry.Entry, error) {
	// Acquire lock so we can't close while Read or Drain is occuring.
	// It will also protect against multiple Close being called at once
	m.readLock.Lock()
	defer m.readLock.Unlock()

	entries := make([]*entry.Entry, 0)

	// Buffer already closed
	if m.closed {
		return entries, nil
	}

	// Mark as closed so any operations after this point won't execute
	m.closed = true

	// Close the buffer channel then drain it
	close(m.buf)
	for range m.buf {
	}

	return entries, nil
}
