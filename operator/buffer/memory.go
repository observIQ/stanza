package buffer

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
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
func (c MemoryBufferConfig) Build(context operator.BuildContext, pluginID string) (Buffer, error) {
	return &MemoryBuffer{}, nil
}

// MemoryBuffer is a buffer that holds all entries in memory until Close() is called,
// at which point it saves the entries into a database. It provides no guarantees about
// lost entries if shut down uncleanly.
type MemoryBuffer struct {
}

// Add adds an entry onto the buffer.
// Is a blocking call if the buffer is full
func (m *MemoryBuffer) Add(ctx context.Context, e *entry.Entry) error {
	return nil
}

// Read reads from the buffer.
// Read will block until the there are MaxChunkSize entries or we have block as long as MachChunkDelay.
func (m *MemoryBuffer) Read(ctx context.Context) ([]*entry.Entry, error) {
	return nil, nil
}

// Drain drains all contents currently in the buffer to the returned entry
func (m *MemoryBuffer) Drain(ctx context.Context) ([]*entry.Entry, error) {
	return nil, nil
}

// Close runs cleanup code for buffer
func (m *MemoryBuffer) Close() error {
	return nil
}
