package buffer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
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
func (c DiskBufferConfig) Build() (Buffer, error) {
	fileFlags := os.O_CREATE | os.O_RDWR
	if c.Sync {
		fileFlags |= os.O_SYNC
	}

	f, err := os.OpenFile(c.Path, fileFlags, 0660)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	var metadata *DiskBufferMetadata

	if fi.Size() >= DiskBufferMetadataBinarySize {
		metadata, err = ReadDiskBufferMetadata(f)
		if err != nil {
			f.Close()
			return nil, err
		}
	} else {
		metadata = NewDiskBufferMetadata()
		err := metadata.Sync(f)
		if err != nil {
			f.Close()
			return nil, err
		}
	}

	endPos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		f.Close()
		return nil, err
	}

	mux := &sync.Mutex{}

	return &DiskBuffer{
		metadata:      metadata,
		end:           endPos,
		f:             f,
		mux:           mux,
		maxSize:       uint64(c.MaxSize),
		maxChunkDelay: c.MaxChunkDelay.Duration,
		maxChunkSize:  c.MaxChunkSize,
		readReady:     sync.NewCond(mux),
		writeReady:    sync.NewCond(mux),
	}, nil
}

// DiskBuffer is a buffer of entries that stores the entries to disk.
// This buffer persists between application restarts.
type DiskBuffer struct {
	metadata *DiskBufferMetadata
	// end is an integer indicating the offset into the buffer (not including metadata) where the end of buffer is
	end int64
	// f is the underlying file that holds the buffer data
	f *os.File
	// mux is a mutex that protects read/write operations to the file
	mux *sync.Mutex
	// maxSize is the maximum number of entry bytes that can be written to the file
	// The max size of the file is actually maxSize + dataOffset
	maxSize uint64

	maxChunkDelay time.Duration
	maxChunkSize  uint
	readReady     *sync.Cond
	writeReady    *sync.Cond
	closed        bool
}

const entryBufInitialSize = 1 << 10

// Add adds an entry onto the buffer.
// Will block if the buffer is full.
func (d *DiskBuffer) Add(ctx context.Context, e *entry.Entry) error {
	d.mux.Lock()
	defer d.mux.Unlock()

	if d.closed {
		return ErrBufferClosed
	}

	bufBytes := make([]byte, 0, entryBufInitialSize)
	bufBytes, err := marshalEntry(bufBytes, e)
	if err != nil {
		return err
	}

	if len(bufBytes) > int(d.maxSize) {
		return errors.New("entry is too large to fit on disk")
	}

	for !d.canFitInBuffer(len(bufBytes)) {
		err := d.compact()
		if err != nil {
			return err
		}

		if d.canFitInBuffer(len(bufBytes)) {
			break
		}

		// Wait for a reader to tell us we can (maybe) write!
		waitCondWithCtx(ctx, d.writeReady)

		select {
		case <-ctx.Done():
			return fmt.Errorf("got context done: %w", ctx.Err())
		default:
		}

		if d.closed {
			return ErrBufferClosed
		}
	}

	curEnd, err := d.f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Assert for sanity
	if curEnd != int64(d.end) {
		return fmt.Errorf("the current eof(%d) was not equal to the expected eof(%d)", curEnd, d.end)
	}

	_, err = d.f.Write(bufBytes)
	if err != nil {
		return err
	}

	d.end += int64(len(bufBytes))

	// Signal to a potentially waiting reader that there is an entry to read
	d.readReady.Signal()

	return nil
}

// Read reads from the buffer.
// Read will block until the there are maxChunkSize entries or the duration maxChunkDelay has passed.
func (d *DiskBuffer) Read(ctx context.Context) ([]*entry.Entry, error) {
	entries := make([]*entry.Entry, 0)
	timer := time.NewTimer(d.maxChunkDelay)
	defer timer.Stop()

	d.mux.Lock()
	defer d.mux.Unlock()

	if d.closed {
		return nil, ErrBufferClosed
	}

	for len(entries) < int(d.maxChunkSize) {
		for d.end <= int64(d.metadata.StartOffset) {
			// No entries to read
			timerDone := waitCondWithCtxAndTimer(ctx, d.readReady, timer)
			if timerDone {
				return entries, nil
			}

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("got context done: %w", ctx.Err())
			default:
			}

			if d.closed {
				return nil, ErrBufferClosed
			}
		}

		// Seek to the start position of the file
		_, err := d.f.Seek(int64(d.metadata.StartOffset), io.SeekStart)
		if err != nil {
			return nil, err
		}

		var entry entry.Entry
		dec := json.NewDecoder(d.f)

		err = dec.Decode(&entry)
		if err != nil {
			return nil, err
		}

		entries = append(entries, &entry)

		decoderOffset := dec.InputOffset()

		// Update start pointer to current position
		d.metadata.StartOffset += uint64(decoderOffset) + 1
		err = d.metadata.Sync(d.f)
		if err != nil {
			return nil, err
		}

		// Signal to the writers that they may be able to write again, since we freed up some space
		d.writeReady.Broadcast()
	}

	return entries, nil
}

// Close runs cleanup code for buffer
func (d *DiskBuffer) Close() ([]*entry.Entry, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	if d.closed {
		return nil, ErrBufferClosed
	}

	err := d.f.Close()
	if err != nil {
		return nil, err
	}

	d.closed = true

	// Tell all the readers/writers to wake up so they don't block while the buffer is closed
	d.readReady.Broadcast()
	d.writeReady.Broadcast()

	return nil, nil
}

// endDiskOffset gives the end offset on disk, taking into account the prologue size.
// func (d *DiskBuffer) endDiskOffset() int64 {
// 	return int64(d.end + filePrologueSize)
// }

// Compact buffer size = 4 Kib
var compactBufferSize int64 = 2 << 12

// compact compacts the file by moving all data backwards to the start of file, then truncating the file such that
// no data is lost.
func (d *DiskBuffer) compact() error {
	if d.metadata.StartOffset <= DiskBufferMetadataBinarySize {
		// StartOffset is at the beginning of the buffer; Nothing to compact
		return nil
	}

	buffer := make([]byte, compactBufferSize)
	var writeOffset int64 = DiskBufferMetadataBinarySize
	var readOffset int64 = int64(d.metadata.StartOffset)
	for {
		n, err := d.f.ReadAt(buffer, readOffset)
		if errors.Is(err, io.EOF) {
			_, err = d.f.WriteAt(buffer[:n], writeOffset)
			if err != nil {
				return err
			}

			err = d.f.Truncate(writeOffset + int64(n))
			if err != nil {
				return err
			}

			d.end = writeOffset + int64(n)

			break
		} else if err != nil {
			return err
		}

		_, err = d.f.WriteAt(buffer, writeOffset)
		if err != nil {
			return err
		}

		writeOffset += compactBufferSize
		readOffset += compactBufferSize
	}

	d.metadata.StartOffset = DiskBufferMetadataBinarySize
	err := d.metadata.Sync(d.f)
	if err != nil {
		return err
	}

	return nil
}

func (d *DiskBuffer) canFitInBuffer(bufLen int) bool {
	return uint64(d.end)+uint64(bufLen)-DiskBufferMetadataBinarySize <= d.maxSize
}

// waitCondWithCtx waits on the given sync.Cond. It can be awakened normally (cond.Signal or cond.Broadcast),
// but it may also be awakened by the context finishing
func waitCondWithCtx(ctx context.Context, cond *sync.Cond) {
	doneChan := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cond.Broadcast()
		case <-doneChan:
		}
	}()

	cond.Wait()
	close(doneChan)
}

// waitCondWithCtxAndTimer waits on the given sync.Cond. It can be awakened normally (cond.Signal or cond.Broadcast),
// but it may also be awakened by the context finishing, or the timer firing.
// Returns true if the timer fired, false otherwise.
func waitCondWithCtxAndTimer(ctx context.Context, cond *sync.Cond, timer *time.Timer) bool {
	doneChan := make(chan struct{})
	wasTimer := make(chan bool)
	go func() {
		timerTriggered := false
		select {
		case <-timer.C:
			cond.Broadcast()
			timerTriggered = true
		case <-ctx.Done():
			cond.Broadcast()
		case <-doneChan:
		}
		wasTimer <- timerTriggered
	}()

	cond.Wait()

	close(doneChan)

	return <-wasTimer
}

func marshalEntry(b []byte, e *entry.Entry) ([]byte, error) {
	buf := bytes.NewBuffer(b)
	enc := json.NewEncoder(buf)
	err := enc.Encode(e)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
