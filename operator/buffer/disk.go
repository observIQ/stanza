package buffer

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

	"github.com/observiq/stanza/entry"
	"golang.org/x/sync/semaphore"
)

var _ Buffer = &DiskBuffer{}

type DiskBuffer struct {
	// Metadata holds information about the current state of the buffered entries
	metadata *Metadata

	// Data is the file that stores the buffered entries
	data *os.File
	sync.Mutex

	// entryAdded is a channel that is notified on every time an entry is added.
	// The integer sent down the channel is the new number of unread entries stored.
	// Readers using ReadWait will listen on this channel, and wait to read until
	// there are enough entries to fill its buffer.
	entryAdded chan int64

	// readerLock ensures that there is only ever one reader listening to the
	// entryAdded channel at a time.
	readerLock sync.Mutex

	// diskSizeSemaphore
	diskSizeSemaphore *semaphore.Weighted

	copyBuffer []byte
}

// NewDiskBuffer creates a new DiskBuffer
func NewDiskBuffer(maxDiskSize int64) *DiskBuffer {
	return &DiskBuffer{
		entryAdded:        make(chan int64, 1),
		copyBuffer:        make([]byte, 1<<16), // TODO benchmark different sizes
		diskSizeSemaphore: semaphore.NewWeighted(maxDiskSize),
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

	info, err := d.data.Stat()
	if err != nil {
		return err
	}

	if ok := d.diskSizeSemaphore.TryAcquire(info.Size()); !ok {
		return fmt.Errorf("current on-disk size is larger than max size")
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

	// Once everything is compacted, we can safely reset all previously read, but
	// unflushed entries to unread
	d.metadata.unreadStartOffset = 0
	d.metadata.unreadCount += int64(len(d.metadata.read))
	d.metadata.read = d.metadata.read[:0]
	return d.metadata.Sync()
}

// Add adds an entry to the buffer, blocking until it is either added or the context
// is cancelled.
func (d *DiskBuffer) Add(ctx context.Context, newEntry *entry.Entry) error {
	var buf bytes.Buffer // TODO pool buffers
	enc := json.NewEncoder(&buf)
	err := enc.Encode(newEntry)
	if err != nil {
		return err
	}

	err = d.diskSizeSemaphore.Acquire(ctx, int64(buf.Len()))
	if err != nil {
		return err
	}

	d.Lock()
	defer d.Unlock()

	// Seek to end of the file
	_, err = d.data.Seek(0, 2)
	if err != nil {
		return err
	}

	_, err = d.data.Write(buf.Bytes())
	if err != nil {
		return err
	}

	d.incrementUnreadCount(1)

	return nil
}

// incrementUnreadCount adds i to the unread count and notifies any callers of
// ReadWait that an entry has been added. The disk buffer lock must be held when
// calling this.
func (d *DiskBuffer) incrementUnreadCount(i int64) {
	d.metadata.unreadCount += i

	// Notify a reader that new entries have been added by either
	// sending on the channel, or updating the value in the channel
	select {
	case <-d.entryAdded:
		d.entryAdded <- d.metadata.unreadCount
	case d.entryAdded <- d.metadata.unreadCount:
	}
}

// ReadWait reads entries from the buffer, waiting until either there are enough entries in the
// buffer to fill dst, or an event is sent down the timeout channel. This amortizes the cost
// of reading from the disk. It returns a function that, when called, marks the read entries as
// flushed, the number of entries read, and an error.
func (d *DiskBuffer) ReadWait(dst []*entry.Entry, timeout <-chan time.Time) (func(), int, error) {
	d.readerLock.Lock()
	defer d.readerLock.Unlock()

	// Wait until the timeout is hit, or there are enough unread entries to fill the destination buffer
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

	return d.Read(dst)
}

// Read copies entries from the disk into the destination buffer. It returns a function that,
// when called, marks the entries as flushed, the number of entries read, and an error.
func (d *DiskBuffer) Read(dst []*entry.Entry) (f func(), i int, err error) {
	d.Lock()
	defer d.Unlock()

	// Return fast if there are no unread entries
	if d.metadata.unreadCount == 0 {
		return func() {}, 0, nil
	}

	// Seek to the start of the range of unread entries
	_, err = d.data.Seek(d.metadata.unreadStartOffset, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("seek to unread: %s", err)
	}

	readCount := min(len(dst), int(d.metadata.unreadCount))
	newRead := make([]*readEntry, readCount)
	dec := json.NewDecoder(d.data)
	entryStartOffset := d.metadata.unreadStartOffset
	for i := 0; i < readCount; i++ {
		var entry entry.Entry
		err := dec.Decode(&entry)
		if err != nil {
			return nil, 0, fmt.Errorf("decode: %s", err)
		}
		dst[i] = &entry

		newRead[i] = &readEntry{
			startOffset: entryStartOffset,
			length:      d.metadata.unreadStartOffset + dec.InputOffset() + 1 - entryStartOffset,
		}
		entryStartOffset = d.metadata.unreadStartOffset + dec.InputOffset() + 1
	}

	d.metadata.read = append(d.metadata.read, newRead...)
	d.metadata.unreadStartOffset = entryStartOffset
	d.metadata.unreadCount -= int64(readCount)
	markFlushed := func() {
		d.Lock()
		for _, entry := range newRead {
			entry.flushed = true
		}
		d.Unlock()
		// TODO auto-compact at some percent space used by flushed
	}

	return markFlushed, readCount, nil
}

// Close flushes the current metadata to disk, then closes the underlying files
func (d *DiskBuffer) Close() error {
	d.Lock()
	defer d.Unlock()

	err := d.metadata.Close()
	if err != nil {
		return err
	}
	return d.data.Close()
}

// Compact removes all flushed entries from disk
func (d *DiskBuffer) Compact() error {
	d.Lock()
	defer d.Unlock()

	// So how does this work? The goal here is to remove all flushed entries from disk,
	// freeing up space for new entries. We do this by going through each entry that has
	// been read and checking if it has been flushed. If it has, we know that space on
	// disk is re-claimable, so we can move unflushed entries into its place.
	//
	// The tricky part is that we can't overwrite any data until we've both safely copied it
	// to its new location and written a copy of the metadata that describes where that data
	// is located on disk. This ensures that, if our process is killed mid-compaction, we will
	// always have a complete, uncorrupted database.
	//
	// We do this by maintaining a "dead range" during compation. The dead range is
	// effectively a range of bytes that can safely be deleted from disk just by shifting
	// everything that comes after it backwards in the file. Then, when we open the disk
	// buffer, the first thing we do is delete the dead range if it exists.
	//
	// To clear out flushed entries, we iterate over all the entries that have been read,
	// finding ranges of either flushed or unflushed entries. If we have a range of flushed
	// entries, we can expand the dead range to include the space those entries took on disk.
	// If we find a range of unflushed entries, we move them to the beginning of the dead range
	// and advance the start of the dead range to the end of the copied bytes.
	//
	// Once we iterate through all the read entries, we should be left with a dead range
	// that's located right before the start of the unread entries. Since we know none of the
	// unread entries need be flushed, we can simply bubble the dead range through the unread
	// entries, then truncate the dead range from the end of the file once we're done.
	//
	// The most important part here is to sync the metadata to disk before overwriting any
	// data. That way, at startup, we know where the dead zone is in the file so we can
	// safely delete it without deleting any live data.
	//
	// Example:
	// (f = flushed byte, r = read byte, u = unread byte, lowercase = dead range)
	//
	// FFFFRRRRFFRRRRRRRUUUUUUUUU // start of compaction
	// ffffRRRRFFRRRRRRRUUUUUUUUU // mark the first flushed range as unread
	// RRRRrrrrFFRRRRRRRUUUUUUUUU // move the read range to the beginning of the dead range
	// RRRRrrrrffRRRRRRRUUUUUUUUU // expand the dead range to include the flushed range
	// RRRRRRRRRRrrrrrrRUUUUUUUUU // move the portion of the next read range that fits into the dead range
	// RRRRRRRRRRRrrrrrrUUUUUUUUU // move the remainder of the read range to start of the dead range
	// RRRRRRRRRRRUUUUUUuuuuuuUUU // move the unread entries that fit into the dead range
	// RRRRRRRRRRRUUUUUUUUUuuuuuu // move the remainder of the unread entries into the dead range
	// RRRRRRRRRRRUUUUUUUUU       // truncate the file to remove the dead range

	m := d.metadata
	if m.deadRangeLength != 0 {
		return fmt.Errorf("cannot compact the disk buffer before removing the dead range")
	}

	for i := 0; i < len(m.read); {
		if m.read[i].flushed {
			// Find the end index of the range of flushed entries
			j := i + 1
			for ; j < len(m.read); j++ {
				if !m.read[i].flushed {
					break
				}
			}

			// Expand the dead range
			rangeSize := onDiskSize(m.read[i:j])
			m.deadRangeLength += rangeSize

			// Update the effective offsets if the dead range is removed
			for _, entry := range m.read[j:] {
				entry.startOffset -= rangeSize
			}

			// Update the effective unreadStartOffset if the dead range is removed
			m.unreadStartOffset -= rangeSize

			// Delete the flushed range from metadata
			m.read = append(m.read[:i], m.read[j:]...)

			// Sync to disk
			err := d.metadata.Sync()
			if err != nil {
				return err
			}
		} else {
			// Find the end index of the range of unflushed entries
			j := i + 1
			for ; j < len(m.read); j++ {
				if m.read[i].flushed {
					break
				}
			}

			// If there is no dead range, no need to move unflushed entries
			if m.deadRangeLength == 0 {
				i = j
				continue
			}

			// Slide the range left, syncing dead range after every chunk
			rangeSize := int(onDiskSize(m.read[i:j]))
			for bytesMoved := 0; bytesMoved < rangeSize; {
				remainingBytes := rangeSize - bytesMoved
				chunkSize := min(int(m.deadRangeLength), remainingBytes)

				// Move the chunk to the beginning of the dead space
				_, err := d.moveRange(
					m.deadRangeStart,
					m.deadRangeLength,
					m.read[i].startOffset+int64(bytesMoved),
					int64(chunkSize),
				)
				if err != nil {
					return err
				}

				// Update the offset of the dead space
				m.deadRangeStart += int64(chunkSize)
				bytesMoved += chunkSize

				// Sync to disk all at once
				err = d.metadata.Sync()
				if err != nil {
					return err
				}
			}

			// Update i
			i = j
		}

	}

	// Bubble the dead space through the unflushed entries, then truncate
	return d.deleteDeadRange()
}

// deleteDeadRange moves the dead range to the end of the file, chunk by chunk,
// so that if it is interrupted, it can just be continued at next startup.
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

	// Truncate the extra space at the end of the file
	err = d.data.Truncate(info.Size() - d.metadata.deadRangeLength)
	if err != nil {
		return err
	}

	err = d.metadata.setDeadRange(0, 0)
	if err != nil {
		return err
	}

	d.diskSizeSemaphore.Release(d.metadata.deadRangeLength)
	return nil
}

// moveRange moves from length2 bytes starting from start2 into the space from start1
// to start1+length1
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

func min(first, second int) int {
	m := first
	if second < first {
		m = second
	}
	return m
}
