package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

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

	// read is the collection of entries that have been read from disk
	read []*diskEntry

	// unreadStartOffset is the offset on disk where the contiguous
	// range of unread entries start
	unreadStartOffset int64

	// unreadCount is the number of unread entries on disk
	unreadCount int64

	// deadRangeStart is file offset of the beginning of the dead range.
	// The dead range is a range of the file that contains unused information
	// and should only exist during a compaction. If this exists on startup,
	// it should be removed as part of the startup compaction.
	deadRangeStart int64

	// deadRangeLength is the length of the dead range
	deadRangeLength int64
}

func OpenMetadata(path string) (*Metadata, error) {
	m := &Metadata{}

	var err error
	m.file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return &Metadata{}, err
	}

	info, err := m.file.Stat()
	if err != nil {
		return &Metadata{}, err
	}

	if info.Size() > 0 {
		err = m.Read(m.file)
		if err != nil {
			return &Metadata{}, fmt.Errorf("read metadata file: %s", err)
		}
	} else {
		m.read = make([]*diskEntry, 0, 1000)
	}

	return m, nil
}

func (m *Metadata) Sync() error {
	var buf bytes.Buffer

	// Serialize to a buffer first so we can do an atomic write operation
	m.Write(&buf)

	n, err := m.file.WriteAt(buf.Bytes(), 0)
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

func (m *Metadata) Close() error {
	err := m.Sync()
	if err != nil {
		return err
	}
	m.file.Close()
	return nil
}

func (m *Metadata) Write(wr io.Writer) {
	_ = binary.Write(wr, binary.LittleEndian, uint64(1))

	_ = binary.Write(wr, binary.LittleEndian, m.deadRangeStart)
	_ = binary.Write(wr, binary.LittleEndian, m.deadRangeLength)

	_ = binary.Write(wr, binary.LittleEndian, m.unreadStartOffset)
	_ = binary.Write(wr, binary.LittleEndian, m.AtomicUnreadCount())

	_ = binary.Write(wr, binary.LittleEndian, uint64(len(m.read)))
	for _, diskEntry := range m.read {
		_ = binary.Write(wr, binary.LittleEndian, diskEntry.flushed)
		_ = binary.Write(wr, binary.LittleEndian, diskEntry.length)
		_ = binary.Write(wr, binary.LittleEndian, diskEntry.startOffset)
	}
}

func (m *Metadata) setDeadRange(start, length int64) error {
	m.deadRangeStart = start
	m.deadRangeLength = length
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, m.deadRangeStart)
	_ = binary.Write(&buf, binary.LittleEndian, m.deadRangeLength)
	_, err := m.file.WriteAt(buf.Bytes(), 8)
	return err
}

func (m *Metadata) AddUnreadCount(n int64) {
	atomic.AddInt64(&m.unreadCount, n)
}

func (m *Metadata) AtomicUnreadCount() int64 {
	return atomic.LoadInt64(&m.unreadCount)
}

func (m *Metadata) Read(r io.Reader) error {
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

	// Read unread info
	err = binary.Read(r, binary.LittleEndian, &m.unreadStartOffset)
	if err != nil {
		return fmt.Errorf("read unread start offset: %s", err)
	}
	err = binary.Read(r, binary.LittleEndian, &m.unreadCount)
	if err != nil {
		return fmt.Errorf("read contiguous count: %s", err)
	}

	// Read read info
	var readCount int64
	err = binary.Read(r, binary.LittleEndian, &readCount)
	if err != nil {
		return fmt.Errorf("read read count: %s", err)
	}
	m.read = make([]*diskEntry, readCount)
	for i := 0; i < int(readCount); i++ {
		newEntry := diskEntry{}
		err = binary.Read(r, binary.LittleEndian, &newEntry.flushed)
		if err != nil {
			return fmt.Errorf("read disk entry flushed: %s", err)
		}

		err = binary.Read(r, binary.LittleEndian, &newEntry.length)
		if err != nil {
			return fmt.Errorf("read disk entry length: %s", err)
		}

		err = binary.Read(r, binary.LittleEndian, &newEntry.startOffset)
		if err != nil {
			return fmt.Errorf("read disk entry start offset: %s", err)
		}

		m.read[i] = &newEntry
	}

	return nil
}

func (m *Metadata) nextFlushedRange() (int, int, bool) {
	start := -1
	end := -1
	for i, diskEntry := range m.read {
		if diskEntry.flushed {
			start = i
			break
		}
	}
	if start == -1 {
		return 0, 0, false
	}

	for i := start; i < len(m.read); i++ {
		if !m.read[i].flushed {
			end = i
			break
		}
	}

	if end == -1 {
		return start, len(m.read), true
	}

	return start, end, true
}

// getCleanableRanges returns the starting and ending indexes of the two ranges  of diskEntries
// where the first range is composed successfully flushed diskEntries and the second
// range is composed of
func getCleanableRanges(searchStart int, entries []*diskEntry) (start1, start2, end2 int) {
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
