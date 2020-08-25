package disk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func (d *DiskBuffer) Read(dst []*entry.Entry) (f func(), i int, err error) {
	d.Lock()
	defer d.Unlock()
	defer func() {
		if err != nil {
			mf, _ := os.Create("/tmp/metadata")
			d.metadata.file.Seek(0, 0)
			io.Copy(mf, d.metadata.file)

			df, _ := os.Create("/tmp/data")
			d.data.Seek(0, 0)
			io.Copy(df, d.data)
		}
	}()

	if d.metadata.unreadCount == 0 {
		return func() {}, 0, nil
	}

	_, err = d.data.Seek(d.metadata.unreadStartOffset, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("seek to unread: %s", err)
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
			return nil, 0, fmt.Errorf("decode: %s", err)
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
	d.metadata.unreadStartOffset = currentOffset + 1 // Add one for the trailing newline
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

	m := d.metadata
	for i := 0; i < len(m.read); {
		if m.read[i].flushed {
			// If the next entry is flushed, find the range of flushed entries, then
			// update the length of the dead space to include the range, delete
			// the flushed entries from metadata, then  sync metadata

			// Find the end index of the slice of flushed entries
			j := i + 1
			for ; j < len(m.read); j++ {
				if !m.read[i].flushed {
					break
				}
			}

			// Expand the dead range
			m.deadRangeLength += onDiskSize(m.read[i:j])

			// Delete the range from metadata
			m.read = append(m.read[:i], m.read[j:]...)
		} else {
			// If the next entry is unflushed, find the range of unflushed entries
			// that can fit completely inside the dead space. Copy those into the dead
			// space, update their offsets, update nextIndex, update the offset of
			// the dead space, then sync metadata

			// Find the end index of the slice of unflushed entries, or the end index
			// of the range that fits inside the dead range
			j := i + 1
			for ; j < len(m.read); j++ {
				if m.read[i].flushed {
					break
				}

				if onDiskSize(m.read[i:j]) > m.deadRangeLength {
					break
				}
			}

			// Move the range into the dead space
			bytesMoved, err := d.moveRange(
				m.deadRangeStart,
				m.deadRangeLength,
				m.read[i].startOffset,
				m.read[j-1].length,
			)
			if err != nil {
				return err
			}

			// Update the offsets of the moved range
			offsetDelta := m.read[i].startOffset - m.deadRangeStart
			for _, diskEntry := range m.read {
				diskEntry.startOffset -= offsetDelta
			}

			// Update the offset of the dead space
			m.deadRangeStart += int64(bytesMoved)

			// Update i
			i = j
		}

		// Sync after every operation
		err := d.metadata.Sync()
		if err != nil {
			return err
		}
	}

	// Bubble the dead space through the unflushed entries, then truncate
	return d.deleteDeadRange()
}

// onDiskSize calculates the size in bytes on disk for a contiguous
// range of diskEntries
func onDiskSize(entries []*diskEntry) int64 {
	if len(entries) == 0 {
		return 0
	}

	last := entries[len(entries)-1]
	return last.startOffset + last.length - entries[0].startOffset
}

func (d *DiskBuffer) deleteDeadRange() error {
	// Exit fast if there is no dead range
	if d.metadata.deadRangeLength == 0 {
		return nil
	}

	for {
		// Replace the range with the proceeding range of bytes
		start := d.metadata.deadRangeStart
		length := d.metadata.deadRangeLength
		n, err := d.moveRange(
			start,
			length,
			start+length,
			length,
		)
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

func (d *DiskBuffer) moveRange(start1, length1, start2, length2 int64) (int, error) {
	if length2 > length1 {
		return 0, fmt.Errorf("cannot move a range into a space smaller than itself")
	}

	readPosition := start2
	writePosition := start1
	bytesRead := 0

	rd := io.LimitReader(d.data, length2)

	eof := false
	for !eof {
		// Seek to last read position
		_, err := d.data.Seek(readPosition, 0)
		if err != nil {
			return 0, err
		}

		// Read a chunk
		n, err := rd.Read(d.copyBuffer)
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			eof = true
		}
		readPosition += int64(n)
		bytesRead += n

		// Write the chunk back into a free region
		_, err = d.data.WriteAt(d.copyBuffer[:n], writePosition)
		if err != nil {
			return 0, err
		}
		writePosition += int64(n)

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
