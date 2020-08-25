package disk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/observiq/carbon/entry"
)

type DiskBuffer struct {
	metadata *Metadata
	data     *os.File
	sync.Mutex

	entryAdded chan int64
	readerLock sync.Mutex

	copyBuffer []byte
}

// NewDiskBuffer creates a new DiskBuffer
func NewDiskBuffer() *DiskBuffer {
	return &DiskBuffer{
		entryAdded: make(chan int64),
		copyBuffer: make([]byte, 1<<16), // TODO benchmark different sizes
	}
}

// Open opens the disk buffer files from a database directory
func (d *DiskBuffer) Open(path string) error {
	var err error
	d.data, err = os.OpenFile(filepath.Join(path, "data"), os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}

	d.metadata, err = OpenMetadata(filepath.Join(path, "metadata"))
	if err != nil {
		return err
	}

	// First, if there is a dead range from a previous incomplete compaction, delete it
	err = d.deleteDeadRange()
	if err != nil {
		return err
	}

	// Compact on open
	err = d.Compact()
	if err != nil {
		return err
	}

	// Once everything is compacted, we can safely reset
	d.metadata.unreadStartOffset = 0
	d.metadata.unreadCount += int64(len(d.metadata.read))
	d.metadata.read = d.metadata.read[:0]
	return d.metadata.Sync()
}

// Add adds an entry to the buffer
func (d *DiskBuffer) Add(ctx context.Context, newEntry *entry.Entry) error {
	d.Lock()
	defer d.Unlock()

	// Seek to end of the file
	_, err := d.data.Seek(0, 2)
	if err != nil {
		return err
	}

	counter := NewCountingWriter(d.data)
	enc := json.NewEncoder(counter)
	err = enc.Encode(newEntry)
	if err != nil {
		return err
	}

	d.metadata.unreadCount++

	// Notify a reader that new entries have been added
	select {
	case d.entryAdded <- d.metadata.unreadCount:
	default:
	}

	return nil
}

func (d *DiskBuffer) ReadWait(dst []*entry.Entry, timeout <-chan time.Time) (func(), int, error) {
	d.readerLock.Lock()
	defer d.readerLock.Unlock()

	// Wait until there are enough entries to do a single read
	if d.metadata.AtomicUnreadCount() < int64(len(dst)) {
	LOOP:
		for {
			select {
			case n := <-d.entryAdded:
				if n >= int64(len(dst)) {
					break LOOP
				}
			case <-timeout:
				break LOOP
			}
		}
	}

	return d.Read(dst)
}

func (d *DiskBuffer) Read(dst []*entry.Entry) (func(), int, error) {
	d.Lock()
	defer d.Unlock()

	if d.metadata.unreadCount == 0 {
		return func() {}, 0, nil
	}

	_, err := d.data.Seek(d.metadata.unreadStartOffset, 0)
	if err != nil {
		return nil, 0, err
	}

	readCount := min(len(dst), int(d.metadata.unreadCount))

	inFlight := make([]*diskEntry, readCount)
	counter := NewCountingReader(d.data)
	currentOffset := d.metadata.unreadStartOffset
	dec := json.NewDecoder(counter)
	for i := 0; i < readCount; i++ {
		var entry entry.Entry
		err := dec.Decode(&entry)
		if err != nil {
			return nil, 0, err
		}
		dst[i] = &entry

		// TODO explanatory comment
		totalRead := counter.BytesRead() - dec.Buffered().(*bytes.Reader).Len()
		inFlight[i] = &diskEntry{
			length:      int64(totalRead) - currentOffset,
			startOffset: currentOffset,
		}
		currentOffset = int64(totalRead)
	}

	d.metadata.read = append(d.metadata.read, inFlight...)
	d.metadata.unreadStartOffset = currentOffset
	d.metadata.unreadCount -= int64(readCount)
	markFlushed := func() {
		d.Lock()
		defer d.Unlock()
		for _, entry := range inFlight {
			entry.flushed = true
		}
	}

	return markFlushed, readCount, nil
}

func (d *DiskBuffer) Close() error {
	d.Lock()
	defer d.Unlock()
	err := d.metadata.Close()
	if err != nil {
		return err
	}
	d.data.Close()
	return nil
}

// Compact removes all flushed entries from disk
func (d *DiskBuffer) Compact() error {
	d.Lock()
	defer d.Unlock()

	deletedBytes := int64(0)
	for {
		start, end, ok := d.metadata.nextFlushedRange()
		if !ok {
			break
		}

		firstFlushed := d.metadata.read[start]
		lastFlushed := d.metadata.read[end-1]

		startOffset := firstFlushed.startOffset
		endOffset := lastFlushed.startOffset + lastFlushed.length
		bytesMoved, err := d.overwriteRange(startOffset, endOffset)
		if err != nil {
			return err
		}

		for _, diskEntry := range d.metadata.read[end:] {
			diskEntry.startOffset -= (endOffset - startOffset)
		}
		// TODO this logic is wrong. We should be moving all read entries to the beginning
		// of the file, then deleting the dead range created in the space before the unread entries

		// Remove range 1 from tracked diskEntries
		d.metadata.read = append(d.metadata.read[:start], d.metadata.read[end:]...)

		d.metadata.deadRangeStart = endOffset
		d.metadata.deadRangeLength = int64(bytesMoved)

		err = d.metadata.Sync()
		if err != nil {
			return err
		}

		d.metadata.unreadStartOffset -= (endOffset - startOffset)
		deletedBytes += (endOffset - startOffset)
	}

	info, err := d.data.Stat()
	if err != nil {
		return err
	}
	return d.data.Truncate(info.Size() - deletedBytes)
}

func (d *DiskBuffer) deleteDeadRange() error {
	// Exit fast if there is no dead range
	if d.metadata.deadRangeLength == 0 {
		return nil
	}

	// Keep atomically overwriting ranges until we're at the end of the file
	for {
		// Replace the range with the proceeding range of bytes
		start := d.metadata.deadRangeStart
		length := d.metadata.deadRangeLength
		n, err := d.overwriteRange(start, start+length)
		if err != nil {
			return err
		}

		// Update the dead range, writing to disk
		err = d.metadata.setDeadRange(start+length, length)
		if err != nil {
			return err
		}

		if int64(n) < d.metadata.deadRangeLength {
			// We're at the end of the file
			break
		}
	}

	info, err := d.data.Stat()
	if err != nil {
		return err
	}

	err = d.data.Truncate(info.Size() - d.metadata.deadRangeLength)
	if err != nil {
		return err
	}

	return d.metadata.setDeadRange(0, 0)
}

func (d *DiskBuffer) overwriteRange(start, end int64) (int, error) {
	readPosition := end
	writePosition := start
	bytesRead := 0
	eof := false
	for !eof {
		// Read a chunk
		n, err := d.data.ReadAt(d.copyBuffer, readPosition)
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			eof = true
		}
		readPosition += int64(n)

		// Write the chunk back into a free region
		_, err = d.data.WriteAt(d.copyBuffer[:n], writePosition)
		if err != nil {
			return 0, err
		}
		writePosition += int64(n)

		bytesRead += n
	}

	return bytesRead, nil
}

type diskEntry struct {
	// A flushed entry is one that has been flushed and is ready
	// to be removed from disk
	flushed bool

	// The number of bytes the entry takes on disk
	length int64

	// The offset in the file where the entry starts
	startOffset int64
}

func min(first, second int) int {
	m := first
	if second < first {
		m = second
	}
	return m
}
