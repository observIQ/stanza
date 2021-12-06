package buffer

import (
	"encoding/binary"
	"io"
)

type DiskBufferMetadata struct {
	// Version is a number indicating the version of the disk buffer file.
	// Currently, only 0 is valid.
	Version uint8
	// StartOffset is a number indicating the read offset in the file, such that File.Seek(os.StartOffset, io.SeekStart)
	// should put the file cursor in the correct position for reading.
	StartOffset uint64
}

const DiskBufferMetadataBinarySize = 9

func NewDiskBufferMetadata() *DiskBufferMetadata {
	return &DiskBufferMetadata{
		Version:     0,
		StartOffset: DiskBufferMetadataBinarySize,
	}
}

// Sync syncs the DiskBufferMetadata to the given file.
func (d *DiskBufferMetadata) Sync(f io.WriteSeeker) error {
	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	return d.Write(f)
}

// Write writes the DiskBufferMetadata to the given io.Writer.
func (d *DiskBufferMetadata) Write(w io.Writer) error {
	buf := make([]byte, DiskBufferMetadataBinarySize)

	buf[0] = d.Version
	binary.BigEndian.PutUint64(buf[1:], d.StartOffset)

	_, err := w.Write(buf)
	return err
}

// Read reads bytes from the given io.Reader into a DiskBufferMetadata struct
func ReadDiskBufferMetadata(r io.Reader) (*DiskBufferMetadata, error) {
	buf := make([]byte, DiskBufferMetadataBinarySize)
	_, err := r.Read(buf)

	if err != nil {
		return nil, err
	}

	return &DiskBufferMetadata{
		Version:     buf[0],
		StartOffset: binary.BigEndian.Uint64(buf[1:]),
	}, nil
}
