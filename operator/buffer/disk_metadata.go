package buffer

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

type fileLike interface {
	io.ReadWriteSeeker
	io.Closer
	Truncate(int64) error
}

// diskBufferMetadata holds metadata relating to the disk buffer,
// that isn't the entry data itself.
type diskBufferMetadata struct {
	// Version is a number indicating the version of the disk buffer file.
	// Currently, only 0 is valid.
	Version uint8 `json:"version"`
	// StartOffset is a number indicating the read offset in the file, such that File.Seek(StartOffset, io.SeekStart)
	// should put the file cursor in the correct position for reading.
	StartOffset int64 `json:"start"`
	// EndOffset is a number indicating the write offset in the file, such that File.Seek(EndOffset, io.SeekStart)
	// should put the file cursor in the correct position for writing.
	EndOffset int64 `json:"end"`
	// Full indicates whether the buffer is full or not
	Full bool `json:"full"`
	// Entries is the number of entries in the buffer
	Entries int64 `json:"entries"`
	// f is the internal file for reading and writing
	f fileLike
	// closed indicates whether the DiskBufferMetadata is closed
	closed bool
	// buf is the buffer used to write
	buf *bytes.Buffer
}

func openDiskBufferMetadata(baseFilePath string, sync bool) (*diskBufferMetadata, error) {
	fileFlags := os.O_CREATE | os.O_RDWR
	if sync {
		fileFlags |= os.O_SYNC
	}

	f, err := os.OpenFile(baseFilePath, fileFlags, 0600)
	if err != nil {
		return nil, err
	}

	bufBytes := make([]byte, 0, metadataBufferSize)
	dbm := &diskBufferMetadata{
		Version:     0,
		StartOffset: 0,
		EndOffset:   0,
		f:           f,
		buf:         bytes.NewBuffer(bufBytes),
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if fi.Size() > 0 {
		err = dbm.readFromDisk()
		if err != nil {
			f.Close()
			return nil, err
		}
	} else {
		err = dbm.sync()
		if err != nil {
			f.Close()
			return nil, err
		}
	}

	return dbm, nil
}

// metadataBufferSize is the initial size of the underlying buffer for disk metadata
const metadataBufferSize = 1 << 10 // 1KiB

// sync syncs the DiskBufferMetadata to the given file.
func (d *diskBufferMetadata) sync() error {
	_, err := d.f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	d.buf.Reset()
	enc := json.NewEncoder(d.buf)
	err = enc.Encode(d)
	if err != nil {
		return err
	}

	_, err = d.f.Write(d.buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (d *diskBufferMetadata) readFromDisk() error {
	_, err := d.f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	enc := json.NewDecoder(d.f)
	return enc.Decode(d)
}

func (d *diskBufferMetadata) close() error {
	if d.closed {
		return nil
	}

	d.closed = true

	err := d.sync()
	if err != nil {
		d.f.Close()
		return err
	}

	return d.f.Close()

}
