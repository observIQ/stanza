package buffer

import (
	"bytes"
	"context"
	"encoding/binary"
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
	metadata Metadata
	data     *os.File
	sync.Mutex

	fastReadChannel chan fastReadRequest
	copyBuffer      []byte
}

// NewDiskBuffer creates a new DiskBuffer
func NewDiskBuffer() *DiskBuffer {
	return &DiskBuffer{
		fastReadChannel: make(chan fastReadRequest),
		copyBuffer:      make([]byte, 1<<16), // TODO benchmark different sizes
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

	// Compact on start so that all our live disk entries are consecutive.
	err = d.Compact()
	if err != nil {
		return err
	}

	return nil
}

// Add adds an entry to the buffer
func (d *DiskBuffer) Add(ctx context.Context, newEntry *entry.Entry) error {
	// TODO use channels instead of locks to play nice with context
	d.Lock()
	defer d.Unlock()

	// Seek to end of the file
	fileEndOffset, err := d.data.Seek(0, 2)
	if err != nil {
		return err
	}

	counter := NewCountingWriter(d.data)
	enc := json.NewEncoder(counter)
	err = enc.Encode(newEntry)
	if err != nil {
		return err
	}

	select {
	case message := <-d.fastReadChannel:
		newDiskEntries := []*diskEntry{newDiskEntry(fileEndOffset, int64(counter.BytesWritten()))}
		err := d.metadata.addReadEntries(newDiskEntries)
		if err != nil {
			return err
		}
		message.response <- newFastReadResponse(newDiskEntries, []*entry.Entry{newEntry})
		d.metadata.unreadStartOffset += int64(counter.BytesWritten())
	default:
		d.metadata.unreadCount += 1
	}

	return nil
}

// addDiskEntries adds the diskEntry metadata both to the in-memory store as well as the
// on disk metadata store

func (d *DiskBuffer) ReadWait(dst []*entry.Entry, timeout <-chan time.Time) (func(), int, error) {
	// Check if there any entries waiting to be read on disk
	f, n, err := d.Read(dst)
	if err != nil {
		return func() {}, 0, err
	}

	// Return early if we've filled the destination slice
	if n == len(dst) {
		return f, n, nil
	}

	// Attempt to fill the remainder of the buffer with entries as they are added
	// so we can avoid disk reads
	fastAdded := make([]*diskEntry, 0, len(dst)-n)
LOOP:
	for n < len(dst) {
		message := newFastReadRequest(len(dst) - n) // TODO pool fast read messages?
		select {
		case <-timeout:
			break LOOP
		case d.fastReadChannel <- message:
			resp := <-message.response
			copy(dst[n:], resp.entries)
			fastAdded = append(fastAdded, resp.diskEntries...)
			n += len(resp.entries)
		}
	}

	markFlushed := func() {
		f()
		d.Lock()
		for _, entry := range fastAdded {
			entry.flushed = true
		}
		d.Unlock()
	}
	return markFlushed, n, nil
}

type CountingWriter struct {
	w io.Writer
	n int
}

func (c CountingWriter) Write(dst []byte) (int, error) {
	n, err := c.w.Write(dst)
	c.n += n
	return n, err
}

func (c CountingWriter) BytesWritten() int {
	return c.n
}

func NewCountingWriter(w io.Writer) CountingWriter {
	return CountingWriter{
		w: w,
	}
}

type CountingReader struct {
	r io.Reader
	n int
}

func (c CountingReader) Read(dst []byte) (int, error) {
	n, err := c.r.Read(dst)
	c.n += n
	return n, err
}

func (c CountingReader) BytesRead() int {
	return c.n
}

func NewCountingReader(r io.Reader) CountingReader {
	return CountingReader{
		r: r,
	}
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
		l := counter.BytesRead() - dec.Buffered().(*bytes.Reader).Len()
		inFlight[i] = &diskEntry{
			length:      int64(l),
			startOffset: currentOffset,
		}
		currentOffset += int64(l)
	}

	d.metadata.read = append(d.metadata.read, inFlight...)
	d.metadata.unreadStartOffset = currentOffset
	markFlushed := func() {
		d.Lock()
		for _, entry := range inFlight {
			entry.flushed = true
		}
		d.Unlock()
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

	// First, if there is a dead range from a previous incomplete compaction, delete it
	err := d.deleteDeadRange()
	if err != nil {
		return err
	}

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

		// Remove range 1 from tracked diskEntries
		d.metadata.read = append(d.metadata.read[:start], d.metadata.read[end:]...)

		d.metadata.deadRangeStart = int64(endOffset)
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

		// Write the chunk back into a free region
		_, err = d.data.WriteAt(d.copyBuffer[:n], writePosition)
		if err != nil {
			return 0, err
		}

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

func newDiskEntry(offset, size int64) *diskEntry {
	return &diskEntry{
		startOffset: offset,
		length:      size,
	}
}

type fastReadRequest struct {
	response chan fastReadResponse
	maxCount int
}

type fastReadResponse struct {
	diskEntries []*diskEntry
	entries     []*entry.Entry
}

func newFastReadRequest(maxCount int) fastReadRequest {
	return fastReadRequest{
		response: make(chan fastReadResponse),
		maxCount: maxCount,
	}
}

func newFastReadResponse(diskEntries []*diskEntry, entries []*entry.Entry) fastReadResponse {
	return fastReadResponse{
		diskEntries: diskEntries,
		entries:     entries,
	}
}

func min(first, second int) int {
	m := first
	if second < first {
		m = second
	}
	return m
}

type Metadata struct {
	// File is a handle to the on-disk metadata store
	//
	// The layout of the file is as follows:
	// - 8 byte DatabaseVersion as LittleEndian uint64
	// - 8 byte DeadRangeStartOffset as LittleEndian uint64
	// - 8 byte DeadRangeLength as LittleEndian uint64
	// - 8 byte UnreadStartOffset as LittleEndian uint64
	// - 8 byte UnreadCount as LittleEndian uint64
	// - 8 byte ReadCount as LittleEndian uint64
	// - Repeated ReadCount times:
	//     - 1 byte Flushed bool as LittleEndian uint8
	//     - 8 byte Length as LittleEndian uint64
	//     - 8 byte StartOffset as LittleEndian uint64
	file *os.File

	// noncontiguous represents the range of entries at the
	// beginning of the data file that aren't guaranteed to be
	// contiguous and unflushed
	read []*diskEntry

	// contiguousStart is the offset in the data file where the
	// block of contiguous, unflushed entries starts.
	unreadStartOffset int64

	// contiguousCount is the number of entries in the contiguous,
	// unflushed block.
	unreadCount int64

	// deadRangeStart is file offset of the beginning of the dead range.
	// The dead range is a range of the file that contains unused information
	// and should only exist during a compaction. If this exists on startup,
	// it should be removed as part of the startup compaction.
	deadRangeStart int64

	// deadRangeLength is the length of the dead range
	deadRangeLength int64
}

func OpenMetadata(path string) (Metadata, error) {
	m := Metadata{}

	var err error
	m.file, err = os.OpenFile(filepath.Join(path, "metadata"), os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return Metadata{}, err
	}

	err = m.Read(m.file)
	if err != nil {
		return Metadata{}, fmt.Errorf("read metadata file: %s", err)
	}

	return m, nil
}

func (m Metadata) Sync() error {
	var buf bytes.Buffer

	// Serialize to a buffer first so we can do an atomic write operation
	m.Write(&buf)

	n, err := m.file.Write(buf.Bytes())
	if err != nil {
		return err
	}

	// Since our on-disk format for metadata self-describes length,
	// it's okay to truncate as a separate operation because an un-truncated
	// file is still readable
	// TODO write a test for this
	err = m.file.Truncate(int64(n))
	if err != nil {
		return err
	}

	return nil
}

func (m Metadata) Close() error {
	err := m.Sync()
	if err != nil {
		return err
	}
	m.file.Close()
	return nil
}

func (m Metadata) Write(wr io.Writer) {
	binary.Write(wr, binary.LittleEndian, uint64(1))

	binary.Write(wr, binary.LittleEndian, m.deadRangeStart)
	binary.Write(wr, binary.LittleEndian, m.deadRangeLength)

	binary.Write(wr, binary.LittleEndian, m.unreadStartOffset)
	binary.Write(wr, binary.LittleEndian, m.unreadCount)

	binary.Write(wr, binary.LittleEndian, uint64(len(m.read)))
	for _, diskEntry := range m.read {
		binary.Write(wr, binary.LittleEndian, diskEntry.flushed)
		binary.Write(wr, binary.LittleEndian, diskEntry.length)
		binary.Write(wr, binary.LittleEndian, diskEntry.startOffset)
	}
}

func (m Metadata) setDeadRange(start, length int64) error {
	m.deadRangeStart = start
	m.deadRangeLength = length
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, m.deadRangeStart)
	binary.Write(&buf, binary.LittleEndian, m.deadRangeLength)
	_, err := m.file.WriteAt(buf.Bytes(), 8)
	return err
}

func (m Metadata) Read(r io.Reader) error {
	// Read version
	var version uint64
	err := binary.Read(r, binary.LittleEndian, &version)
	if err != nil {
		return fmt.Errorf("failed to read version: %s", err)
	}

	// Read dead range
	err = binary.Read(r, binary.LittleEndian, &m.deadRangeStart)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &m.deadRangeLength)
	if err != nil {
		return err
	}

	// Read contiguous
	binary.Read(r, binary.LittleEndian, &m.unreadStartOffset)
	err = binary.Read(r, binary.LittleEndian, &version)
	if err != nil {
		return fmt.Errorf("failed to read contiguous start offset: %s", err)
	}
	err = binary.Read(r, binary.LittleEndian, &m.unreadCount)
	if err != nil {
		return fmt.Errorf("failed to read contiguous count: %s", err)
	}

	// Read noncontiguous
	var readCount uint64
	binary.Read(r, binary.LittleEndian, &readCount)
	m.read = make([]*diskEntry, readCount)
	for i := 0; i < int(readCount); i++ {
		newEntry := diskEntry{}
		err = binary.Read(r, binary.LittleEndian, &newEntry.flushed)
		if err != nil {
			return fmt.Errorf("failed to read disk entry flushed: %s", err)
		}

		binary.Read(r, binary.LittleEndian, &newEntry.length)
		if err != nil {
			return fmt.Errorf("failed to read disk entry length: %s", err)
		}

		binary.Read(r, binary.LittleEndian, &newEntry.startOffset)
		if err != nil {
			return fmt.Errorf("failed to read disk entry start offset: %s", err)
		}
	}

	return nil
}

func (m Metadata) nextFlushedRange() (int, int, bool) {
	start := -1
	end := -1

	for i, entry := range m.read {
		if entry.flushed {
			start = i
			break
		}
	}

	for i, entry := range m.read[start:] {
		if !entry.flushed {
			break
		}
		end = i
	}

	if start == -1 || end == -1 {
		return 0, 0, false
	}

	return start, end, true
}

func (m Metadata) addReadEntries(entries []*diskEntry) error {
	var buf bytes.Buffer
	for _, diskEntry := range entries {
		binary.Write(&buf, binary.LittleEndian, diskEntry.flushed)
		binary.Write(&buf, binary.LittleEndian, diskEntry.length)
		binary.Write(&buf, binary.LittleEndian, diskEntry.startOffset)
	}

	_, err := m.file.WriteAt(buf.Bytes(), int64(48+len(m.read)*17))
	if err != nil {
		return err
	}

	m.read = append(m.read, entries...)

	_, err = m.file.Seek(40, 0)
	if err != nil {
		return err
	}

	return binary.Write(m.file, binary.LittleEndian, int64(len(m.read)))
}
