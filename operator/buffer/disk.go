package buffer

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

var _ Buffer = (*DiskBuffer)(nil)

// DiskBufferConfig is a configuration struct for a DiskBuffer
type DiskBufferConfig struct {
	Type string `json:"type" yaml:"type"`

	// MaxSize is the maximum size in bytes of the data file on disk
	MaxSize helper.ByteSize `json:"max_size" yaml:"max_size"`

	// Path is a path to a directory which contains the data and metadata files
	Path string `json:"path" yaml:"path"`

	// Sync indicates whether to open the files with O_SYNC. If this is set to false,
	// in cases like power failures or unclean shutdowns, logs may be lost or the
	// database may become corrupted.
	Sync bool `json:"sync" yaml:"sync"`

	MaxChunkDelay helper.Duration `json:"max_delay"   yaml:"max_delay"`
	MaxChunkSize  uint            `json:"max_chunk_size" yaml:"max_chunk_size"`
}

// NewDiskBufferConfig creates a new default disk buffer config
func NewDiskBufferConfig() *DiskBufferConfig {
	return &DiskBufferConfig{
		Type:          "disk",
		MaxSize:       1 << 32, // 4GiB
		Sync:          true,
		MaxChunkDelay: helper.NewDuration(time.Second),
		MaxChunkSize:  1000,
	}
}

// Build creates a new Buffer from a DiskBufferConfig
func (c DiskBufferConfig) Build(context operator.BuildContext, _ string) (Buffer, error) {
	return &DiskBuffer{}, nil
}

// TODO add comment
type DiskBuffer struct {
}

// Add adds an entry onto the buffer.
// Is a blocking call if the buffer is full
func (m *DiskBuffer) Add(ctx context.Context, e *entry.Entry) error {
	return nil
}

// Read reads from the buffer.
// Read will block until the there are MaxChunkSize entries or we have block as long as MachChunkDelay.
func (m *DiskBuffer) Read(ctx context.Context) ([]*entry.Entry, error) {
	return nil, nil
}

// Drain drains all contents currently in the buffer to the returned entry
func (m *DiskBuffer) Drain(ctx context.Context) ([]*entry.Entry, error) {
	return nil, nil
}

// Close runs cleanup code for buffer
func (m *DiskBuffer) Close() error {
	return nil
}
