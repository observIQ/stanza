package buffer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"golang.org/x/sync/semaphore"
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
func (c DiskBufferConfig) Build() (Buffer, error) {
	if c.Path == "" {
		return nil, os.ErrNotExist
	}

	bufferFilePath := filepath.Join(c.Path, "buffer")
	metadataFilePath := filepath.Join(c.Path, "metadata.json")

	metadata, err := openDiskBufferMetadata(metadataFilePath, c.Sync)
	if err != nil {
		return nil, err
	}

	cf, err := openCircularFile(bufferFilePath, c.Sync, int64(c.MaxSize), metadata)
	if err != nil {
		metadata.Close()
		return nil, err
	}

	sem := semaphore.NewWeighted(int64(c.MaxSize))
	acquired := sem.TryAcquire(cf.len())
	if !acquired {
		metadata.Close()
		cf.Close()
		return nil, errors.New("failed to acquire buffer length for semaphore")
	}

	return &DiskBuffer{
		metadata:      metadata,
		cf:            cf,
		cfMux:         &sync.Mutex{},
		writerSem:     sem,
		readerSem:     newGreedyCountingSemaphore(metadata.Entries),
		maxSize:       int64(c.MaxSize),
		maxChunkDelay: c.MaxChunkDelay.Duration,
		maxChunkSize:  c.MaxChunkSize,
		closedMux:     &sync.RWMutex{},
	}, nil
}

// DiskBuffer is a buffer of entries that stores the entries to disk.
// This buffer persists between application restarts.
type DiskBuffer struct {
	metadata *diskBufferMetadata
	// f is the underlying byte buffer for the disk buffer
	cf        *circularFile
	cfMux     *sync.Mutex
	writerSem *semaphore.Weighted
	readerSem *greedyCountingSemaphore
	// maxSize is the maximum number of entry bytes that can be written to the buffer file.
	maxSize int64
	// closed is a bool indicating if the buffer is closed
	closed    bool
	closedMux *sync.RWMutex

	maxChunkDelay time.Duration
	maxChunkSize  uint
}

// Add adds an entry onto the buffer.
// Will block if the buffer is full
func (d *DiskBuffer) Add(ctx context.Context, e *entry.Entry) error {
	d.closedMux.RLock()
	defer d.closedMux.RUnlock()

	if d.closed {
		return ErrBufferClosed
	}

	bufBytes, err := marshalEntry(e)
	if err != nil {
		return err
	}

	// We cannot fit something into the file that exceeds the size of the file
	if len(bufBytes) > int(d.maxSize) {
		return ErrEntryTooLarge
	}

	err = d.writerSem.Acquire(ctx, int64(len(bufBytes)))
	if err != nil {
		return err
	}

	d.cfMux.Lock()
	defer d.cfMux.Unlock()

	_, err = d.cf.Write(bufBytes)
	if err != nil {
		return err
	}

	// Update metadata with current buffer state
	d.metadata.Entries += 1
	d.cf.SyncToMetadata(d.metadata)
	err = d.metadata.SyncToDisk()
	if err != nil {
		return err
	}

	// Increment the counting semaphore to signal readers that an entry is available
	d.readerSem.Increment()

	return nil
}

// Read reads from the buffer.
// Read will block until the there are maxChunkSize entries or the duration maxChunkDelay has passed.
func (d *DiskBuffer) Read(ctx context.Context) ([]*entry.Entry, error) {
	d.closedMux.RLock()
	defer d.closedMux.RUnlock()

	if d.closed {
		return nil, ErrBufferClosed
	}

	// The reader gains ownership of n entries here.
	n := d.readerSem.AcquireAtMost(ctx, d.maxChunkDelay, int64(d.maxChunkSize))

	if n == 0 {
		return nil, ctx.Err()
	}

	entries := make([]*entry.Entry, 0, n)

	d.cfMux.Lock()
	defer d.cfMux.Unlock()

	dec := json.NewDecoder(d.cf)

	for i := int64(0); i < n; i++ {
		var entry entry.Entry
		err := dec.Decode(&entry)
		if err != nil {
			return entries, err
		}

		entries = append(entries, &entry)
	}

	decoderOffset := dec.InputOffset()
	d.cf.Discard(decoderOffset)
	d.writerSem.Release(decoderOffset)

	// Update metadata with current buffer state
	d.metadata.Entries -= n
	d.cf.SyncToMetadata(d.metadata)
	err := d.metadata.SyncToDisk()

	return entries, err
}

// Close runs cleanup code for buffer.
func (d *DiskBuffer) Close() ([]*entry.Entry, error) {
	d.closedMux.Lock()
	defer d.closedMux.Unlock()

	if d.closed {
		return nil, nil
	}

	d.closed = true

	err := d.cf.Close()
	if err != nil {
		d.metadata.Close()
		return nil, err
	}

	return nil, d.metadata.Close()
}

// marshalEntry marshals the given entry into a byte slice.
// It returns a byte slice containing the marshalled entry.
func marshalEntry(e *entry.Entry) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(e)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
