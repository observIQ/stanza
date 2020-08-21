package buffer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/observiq/carbon/entry"
)

type DiskBuffer struct {
	diskEntries        []*diskEntry
	nextFlushableIndex int

	data     *os.File
	metadata *os.File
	sync.Mutex

	copyBuffer []byte
}

// NewDiskBuffer creates a new DiskBuffer
func NewDiskBuffer() *DiskBuffer {
	return &DiskBuffer{
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

	d.metadata, err = os.OpenFile(filepath.Join(path, "metadata"), os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}

	err = d.loadMetadata()
	if err != nil {
		return err
	}

	// Compact on start so that all our live disk entries are consecutive.
	err = d.Compact()
	if err != nil {
		return err
	}

	// On startup, nextFlushableIndex is not set correctly by Compact,
	// so we compact, then recalculate it
	// TODO try to make this intrinsic
	d.nextFlushableIndex = 0
	for i := len(d.diskEntries) - 1; i >= 0; i-- {
		if d.diskEntries[i].flushed {
			d.nextFlushableIndex = i + 1
		}
	}
	return nil
}

// BatchAdd adds a slice of entry.Entry to the buffer
func (d *DiskBuffer) BatchAdd(ctx context.Context, entries []*entry.Entry) error {
	// TODO use channels instead of locks to play nice with context
	d.Lock()
	defer d.Unlock()

	// Seek to end of the file
	fileEndOffset, err := d.data.Seek(0, 2)
	if err != nil {
		return err
	}

	wr := bufio.NewWriter(d.data)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	newDiskEntries := make([]*diskEntry, 0, len(entries))
	currentOffset := fileEndOffset
	for _, entry := range entries {
		err = enc.Encode(entry)
		if err != nil {
			return err
		}

		n, err := buf.WriteTo(wr)
		if err != nil {
			return err
		}

		newDiskEntries = append(newDiskEntries, NewDiskEntry(currentOffset, n))
		currentOffset += n
	}

	err = d.addDiskEntries(newDiskEntries)
	if err != nil {
		return err
	}

	return wr.Flush()
}

// Add adds an entry to the buffer
func (d *DiskBuffer) Add(ctx context.Context, entry *entry.Entry) error {
	// TODO use channels instead of locks to play nice with context
	d.Lock()
	defer d.Unlock()

	// Seek to end of the file
	fileEndOffset, err := d.data.Seek(0, 2)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err = enc.Encode(entry)
	if err != nil {
		return err
	}

	n, err := d.data.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return d.addDiskEntries([]*diskEntry{NewDiskEntry(fileEndOffset, int64(n))})
}

// addDiskEntries adds the diskEntry metadata both to the in-memory store as well as the
// on disk metadata store
func (d *DiskBuffer) addDiskEntries(entries []*diskEntry) error {
	d.diskEntries = append(d.diskEntries, entries...)

	var buf bytes.Buffer
	binDiskEntry := [9]byte{}
	for _, diskEntry := range entries {
		diskEntry.marshalBinary(binDiskEntry[:])
		_, err := buf.Write(binDiskEntry[:])
		if err != nil {
			return err
		}
	}

	_, err := d.metadata.Seek(0, 2)
	if err != nil {
		return err
	}

	_, err = d.metadata.Write(buf.Bytes())
	if err != nil {
		return err
	}

	count := [8]byte{}
	binary.LittleEndian.PutUint64(count[:], uint64(len(d.diskEntries)))
	_, err = d.metadata.WriteAt(count[:], 0)

	return err
}

// syncMetadata writes the in-memory metadata to disk. It should only be called
// when the buffer's lock is held
func (d *DiskBuffer) syncMetadata() error {
	var buf bytes.Buffer

	count := [8]byte{}
	binary.LittleEndian.PutUint64(count[:], uint64(len(d.diskEntries)))
	buf.Write(count[:])

	binDiskEntry := [9]byte{}
	for _, diskEntry := range d.diskEntries {
		diskEntry.marshalBinary(binDiskEntry[:])
		_, err := buf.Write(binDiskEntry[:])
		if err != nil {
			return err
		}
	}

	// Write full buffer
	_, err := d.metadata.WriteAt(buf.Bytes(), 0)
	if err != nil {
		return err
	}

	// Truncate file
	return d.metadata.Truncate(int64(9*len(d.diskEntries)) + 8)
}

// loadMetadata loads the buffer's metadata from disk
func (d *DiskBuffer) loadMetadata() error {
	// Seek to beginning of file
	_, err := d.metadata.Seek(0, 0)
	if err != nil {
		return err
	}

	rd := bufio.NewReader(d.metadata)

	// The first 8 bytes are the number of entries
	countBytes := [8]byte{}
	_, err = rd.Read(countBytes[:])
	if err != nil {
		if err == io.EOF {
			d.diskEntries = make([]*diskEntry, 0, 100)
			return nil
		}
		return err
	}
	count := binary.LittleEndian.Uint64(countBytes[:])
	d.diskEntries = make([]*diskEntry, 0, count)

	binDiskEntry := [9]byte{}
	currentOffset := int64(0)
	for i := uint64(0); i < count; i++ {
		_, err := rd.Read(binDiskEntry[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		var entry diskEntry
		entry.unmarshalBinary(binDiskEntry[:])
		entry.startOffset = currentOffset
		currentOffset += entry.onDiskSize
		d.diskEntries = append(d.diskEntries, &entry)
	}

	return nil
}

func (d *DiskBuffer) Read(dst []*entry.Entry) (func(), int, error) {
	// A note on file layout and assumptions:
	// With our file layout, we guarantee that all flushable entries are
	// contiguous and at the end of the file. Once an entry is read, there
	// is no way to mark the entry as unread. If it cannot be flushed,
	// the reader must handle erroring or retrying.
	//
	// Since we know that all flushable entries are at the end, we can just
	// track the index of the next entry ready to be read, then read from
	// disk consecutively until our destination buffer is full. This allows
	// us to do large, buffered reads for efficiency.
	//
	// The only time flushable entries are not guaranteed to be at the end is
	// at startup, when there may be entries that have been read, but not marked
	// as flushed. This is why we run a compaction when we create a buffer object
	// from an existing file.
	d.Lock()
	defer d.Unlock()

	if d.nextFlushableIndex == len(d.diskEntries) {
		return func() {}, 0, nil
	}

	// Seek to the beginning of the next entry to flush
	_, err := d.data.Seek(d.diskEntries[d.nextFlushableIndex].startOffset, 0)
	if err != nil {
		return nil, 0, err
	}

	// Read count is minimum of the length of the destination
	// slice and the number of entries available
	flushCount := len(dst)
	readyToFlush := len(d.diskEntries) - d.nextFlushableIndex
	if readyToFlush < flushCount {
		flushCount = readyToFlush
	}

	dec := json.NewDecoder(d.data)
	for i := 0; i < flushCount; i++ {
		var entry entry.Entry
		err := dec.Decode(&entry)
		if err != nil {
			return nil, 0, err
		}
		dst[i] = &entry
	}

	inFlight := make([]*diskEntry, flushCount)
	copy(inFlight, d.diskEntries[d.nextFlushableIndex:d.nextFlushableIndex+flushCount])
	d.nextFlushableIndex += flushCount
	markFlushed := func() {
		for _, entry := range inFlight {
			entry.flushed = true
		}
	}

	return markFlushed, flushCount, nil
}

func (d *DiskBuffer) Close() error {
	d.Lock()
	defer d.Unlock()
	err := d.syncMetadata()
	if err != nil {
		return err
	}
	d.data.Close()
	d.metadata.Close()
	return nil
}

// Compact removes all flushed entries from disk
func (d *DiskBuffer) Compact() error {
	// Overview of the compaction algorithm:
	// The first step is to find two ranges of diskEntries. The first range
	// is composed of all flushed entries. The second range immediately follows
	// the first and is composed of all unflushed entries.
	//
	// Additionally, the on-disk size of the second range is strictly smaller than the on-disk
	// size of the first range. This is so we can guarantee that our operations are (loosely) atomic --
	// we will never overwrite any of the second (live) range until the operation is complete.
	//
	// Once we have our ranges, we copy the second range into the space of the first range.
	// This is done in chunks so we can reuse a large buffer. Once we have successfully copied
	// the full second range, we update the offsets for the unflushed diskEntries then write
	// the updated offsets to disk.
	//
	// This process is repeated until there are no more flushed entries on disk.
	// At this point, the file is truncated to the end of the last live entry to
	// return disk space to the OS.

	searchStart := 0
	for {
		// Find range 1 and range 2
		start1, start2, end2 := getRanges(searchStart, d.diskEntries)
		if start1 == start2 || start2 == end2 {
			// Finish when there are no flushed entries on disk
			// TODO make it more clear why this is true
			break
		}

		// In the next iteration, start searching for ranges
		// beginning at the end of range 2 (the index changes
		// after deleting range 1)
		searchStart = end2 - (start2 - start1)

		// Copy range 2 into the space of range 1
		offset, err := d.overwriteRange(d.diskEntries[start1:start2], d.diskEntries[start2:end2])
		if err != nil {
			return err
		}

		// Update the offsets of range 2
		for _, diskEntry := range d.diskEntries[start2:end2] {
			diskEntry.startOffset -= offset
		}

		// Remove range 1 from tracked diskEntries
		d.diskEntries = append(d.diskEntries[:start1], d.diskEntries[start2:]...)

		err = d.syncMetadata()
		if err != nil {
			return err
		}

		d.nextFlushableIndex -= start2 - start1
	}

	return d.data.Truncate(onDiskSize(d.diskEntries))
}

func (d *DiskBuffer) overwriteRange(range1, range2 []*diskEntry) (int64, error) {
	reader := io.LimitedReader{
		R: d.data,
		N: onDiskSize(range2),
	}

	readPosition := range2[0].startOffset
	writePosition := range1[0].startOffset
	movedOffset := readPosition - writePosition
	eof := false
	for !eof {
		// Seek to the current read position
		_, err := d.data.Seek(readPosition, 0)
		if err != nil {
			return 0, err
		}

		// Read a chunk
		n, err := reader.Read(d.copyBuffer)
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			eof = true
		}

		// Write the chunk back into a free region
		_, err = d.data.WriteAt(d.copyBuffer[:n], writePosition)
		if err != nil {
			return 0, err
		}
	}

	return movedOffset, nil
}

// getRanges returns the starting and ending indexes of the two ranges  of diskEntries
// where the first range is composed successfully flushed diskEntries and the second
// range is composed of
func getRanges(searchStart int, entries []*diskEntry) (start1, start2, end2 int) {
	// search for the first flushed entry in range 1
	for start1 = searchStart; start1 < len(entries); start1++ {
		if entries[start1].flushed {
			break
		}
	}

	// search for the last flushed entry in range 1
	for start2 = start1; start2 < len(entries); start2++ {
		if !entries[start2].flushed {
			break
		}
	}

	range1DiskSize := onDiskSize(entries[start1:start2])

	// search for the last unflushed entry, or the last entry that will allow
	// range2 to fit inside the space of range 1
	for end2 = start2; end2 < len(entries); end2++ {
		if entries[end2].flushed {
			break
		}

		range2DiskSize := onDiskSize(entries[start2:end2])
		if range2DiskSize > range1DiskSize {
			break
		}
	}

	return start1, start2, end2
}

// onDiskSize calculates the size in bytes on disk for a contiguous
// range of diskEntries
func onDiskSize(entries []*diskEntry) int64 {
	if len(entries) == 0 {
		return 0
	}

	last := entries[len(entries)-1]
	return last.startOffset + last.onDiskSize - entries[0].startOffset
}

type diskEntry struct {
	// A flushed entry is one that has been flushed and is ready
	// to be removed from disk
	flushed bool

	// The offset in the file where the entry starts
	startOffset int64

	// The number of bytes the entry takes on disk
	onDiskSize int64
}

func (d *diskEntry) marshalBinary(dst []byte) {
	if d.flushed {
		dst[0] = 1
	} else {
		dst[0] = 0
	}
	binary.LittleEndian.PutUint64(dst[1:9], uint64(d.onDiskSize))
}

func (d *diskEntry) unmarshalBinary(src []byte) {
	if src[0] == 1 {
		d.flushed = true
	}
	d.onDiskSize = int64(binary.LittleEndian.Uint64(src[1:9]))
}

func NewDiskEntry(offset, size int64) *diskEntry {
	return &diskEntry{
		startOffset: offset,
		onDiskSize:  size,
	}
}
