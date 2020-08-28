package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Metadata struct {
	// File is a handle to the on-disk metadata store
	//
	// The layout of the file is as follows:
	// - 8 byte DatabaseVersion as LittleEndian int64
	// - 8 byte DeadRangeStartOffset as LittleEndian int64
	// - 8 byte DeadRangeLength as LittleEndian int64
	// - 8 byte UnreadStartOffset as LittleEndian int64
	// - 8 byte UnreadCount as LittleEndian int64
	// - 8 byte ReadCount as LittleEndian int64
	// - Repeated ReadCount times:
	//     - 1 byte Flushed bool LittleEndian
	//     - 8 byte Length as LittleEndian uint64
	//     - 8 byte StartOffset as LittleEndian uint64
	file *os.File

	// read is the collection of entries that have been read
	read []*readEntry

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
		m.read = make([]*readEntry, 0, 1000)
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
	_ = binary.Write(wr, binary.LittleEndian, int64(1))

	_ = binary.Write(wr, binary.LittleEndian, m.deadRangeStart)
	_ = binary.Write(wr, binary.LittleEndian, m.deadRangeLength)

	_ = binary.Write(wr, binary.LittleEndian, m.unreadStartOffset)
	_ = binary.Write(wr, binary.LittleEndian, m.unreadCount)

	_ = binary.Write(wr, binary.LittleEndian, int64(len(m.read)))
	for _, readEntry := range m.read {
		_ = binary.Write(wr, binary.LittleEndian, readEntry.flushed)
		_ = binary.Write(wr, binary.LittleEndian, readEntry.length)
		_ = binary.Write(wr, binary.LittleEndian, readEntry.startOffset)
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

func (m *Metadata) Read(r io.Reader) error {
	// Read version
	var version int64
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
	m.read = make([]*readEntry, readCount)
	for i := 0; i < int(readCount); i++ {
		newEntry := readEntry{}
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

// readEntry is a struct holding metadata about read entries
type readEntry struct {
	// A flushed entry is one that has been flushed and is ready
	// to be removed from disk
	flushed bool

	// The number of bytes the entry takes on disk
	length int64

	// The offset in the file where the entry starts
	startOffset int64
}

// onDiskSize calculates the size in bytes on disk for a contiguous
// range of diskEntries
func onDiskSize(entries []*readEntry) int64 {
	if len(entries) == 0 {
		return 0
	}

	last := entries[len(entries)-1]
	return last.startOffset + last.length - entries[0].startOffset
}
