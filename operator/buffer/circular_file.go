package buffer

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/multierr"
)

// circularFile is a io.ReadWriteCloser that writes to a fixed length file, such that
// it wraps around to the beginning when reaching the end.
// Methods on this struct are not thread-safe and require additional synchronization.
type circularFile struct {
	Start      int64
	ReadPtr    int64
	End        int64
	Size       int64
	Full       bool
	f          fileLike
	seekedRead bool
	seekedEnd  bool
	closed     bool
	readPtrEnd bool
}

var _ io.ReadWriteCloser = (*circularFile)(nil)

func openCircularFile(filePath string, sync bool, size int64) (*circularFile, error) {
	fileFlags := os.O_CREATE | os.O_RDWR
	if sync {
		fileFlags |= os.O_SYNC
	}

	f, err := os.OpenFile(filePath, fileFlags, 0600)
	if err != nil {
		return nil, err
	}

	// Make sure that the file, if it already existed, is actually the correct size.
	fsize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		fCloseErr := f.Close()
		return nil, multierr.Combine(
			err,
			fCloseErr,
		)
	}

	if fsize == 0 {
		err := f.Truncate(size)
		if err != nil {
			fCloseErr := f.Close()
			return nil, multierr.Combine(
				err,
				fCloseErr,
			)
		}
	} else if fsize != size {
		fCloseErr := f.Close()
		return nil,
			multierr.Combine(
				fmt.Errorf("configured size (%d) does not match current on-disk size (%d)", size, fsize),
				fCloseErr,
			)
	}

	return &circularFile{
		Size: size,
		f:    f,
	}, nil
}

func (cf *circularFile) Close() error {
	if cf.closed {
		return nil
	}

	cf.closed = true

	err := cf.f.Close()
	if err != nil {
		return err
	}

	return nil
}

// Read reads from the circular file into p, up to len(p) bytes.
// The contents read are not discarded from the buffer; An independent read pointer
// is maintained, which can be reset to the start of the buffer using ResetReadOffset()
func (cf *circularFile) Read(p []byte) (int, error) {
	if cf.closed {
		return 0, ErrBufferClosed
	}

	if len(p) == 0 {
		return 0, nil
	}

	err := cf.seekReadStart()
	if err != nil {
		return 0, err
	}

	var totalBytesToRead int64
	var isEOF bool
	if cf.ReadBytesLeft() < int64(len(p)) {
		totalBytesToRead = cf.ReadBytesLeft()
		isEOF = true
	} else {
		totalBytesToRead = int64(len(p))
		isEOF = false
	}

	var firstReadBytes int64
	if totalBytesToRead+cf.ReadPtr <= cf.Size {
		firstReadBytes = totalBytesToRead
	} else {
		firstReadBytes = cf.Size - cf.ReadPtr
	}

	n, err := cf.f.Read(p[:firstReadBytes])
	cf.ReadPtr = (cf.ReadPtr + int64(n)) % cf.Size
	if cf.ReadPtr == 0 {
		cf.seekedRead = false
	}

	if err != nil {
		return n, err
	}

	if firstReadBytes != totalBytesToRead {
		err = cf.seekReadStart()
		if err != nil {
			return n, err
		}

		n2, err := cf.f.Read(p[firstReadBytes:])
		cf.ReadPtr += int64(n2)
		if err != nil {
			return n + n2, err
		}
	}

	if cf.ReadPtr == cf.End && n != 0 {
		cf.readPtrEnd = true
	}

	if isEOF {
		return int(totalBytesToRead), io.EOF
	}

	return int(totalBytesToRead), nil
}

// Write writes to the CircularFile.
// If the end of the file has been reached and no more bytes may be written,
// io.EOF is returned as an error, as well as the number of bytes that were successfully
// written.
func (cf *circularFile) Write(p []byte) (int, error) {
	if cf.closed {
		return 0, ErrBufferClosed
	}

	if len(p) == 0 {
		return 0, nil
	}

	err := cf.seekEnd()
	if err != nil {
		return 0, err
	}

	var totalBytesToWrite int64
	var isEOF bool
	if cf.writeBytesLeft() < int64(len(p)) {
		totalBytesToWrite = cf.writeBytesLeft()
		isEOF = true
	} else {
		totalBytesToWrite = int64(len(p))
		isEOF = false
	}

	var firstWriteBytes int64
	if totalBytesToWrite+cf.End <= cf.Size {
		firstWriteBytes = totalBytesToWrite
	} else {
		firstWriteBytes = cf.Size - cf.End
	}

	n, err := cf.f.Write(p[:firstWriteBytes])
	// Modulus is used here because rb.end may equal rb.size
	cf.End = (cf.End + int64(n)) % cf.Size
	if cf.End == 0 {
		cf.seekedEnd = false
	}

	if n > 0 {
		cf.readPtrEnd = false
	}

	if err != nil {
		return n, err
	}

	if firstWriteBytes != totalBytesToWrite {
		err = cf.seekEnd()
		if err != nil {
			return n, err
		}

		n2, err := cf.f.Write(p[firstWriteBytes:])
		cf.End += int64(n2)

		if n2 > 0 {
			cf.readPtrEnd = false
		}

		if err != nil {
			return n + n2, err
		}
	}

	if isEOF {
		cf.Full = true
		return int(totalBytesToWrite), io.EOF
	}

	if cf.Start == cf.End {
		cf.Full = true
	}

	return int(totalBytesToWrite), nil
}

func (cf *circularFile) len() int64 {
	if cf.Full {
		return cf.Size
	}

	if cf.Start <= cf.End {
		return cf.End - cf.Start
	} else {
		return cf.End + (cf.Size - cf.Start)
	}
}

func (cf *circularFile) ReadBytesLeft() int64 {
	if cf.readPtrEnd {
		return 0
	}

	if cf.Full && cf.ReadPtr == cf.End {
		return cf.Size
	}

	if cf.ReadPtr <= cf.End {
		return cf.End - cf.ReadPtr
	} else {
		return cf.End + (cf.Size - cf.ReadPtr)
	}
}

func (cf *circularFile) writeBytesLeft() int64 {
	return cf.Size - cf.len()
}

// discard removes n bytes from the start end of the circular file.
// This resets the internal read pointer to be pointed to start.
// If n is greater than the length of the buffer, the buffer is truncated to a length of 0.
func (cf *circularFile) discard(n int64) {
	cf.seekedRead = false
	if n == 0 {
		cf.ReadPtr = cf.Start
		cf.readPtrEnd = !cf.Full && (cf.ReadPtr == cf.End)
		return
	}

	if n > cf.len() {
		cf.Start = cf.End
	} else {
		cf.Start = (cf.Start + n) % cf.Size
	}

	cf.ReadPtr = cf.Start
	cf.Full = false
	cf.readPtrEnd = cf.ReadPtr == cf.End
}

// seekReadStart seeks the underlying file to readPtr.
// We reduce sync calls by keeping track of whether we are at the readPtr
// or at the end pointer.
func (cf *circularFile) seekReadStart() error {
	if cf.seekedRead {
		return nil
	}

	_, err := cf.f.Seek(cf.ReadPtr, io.SeekStart)
	if err != nil {
		cf.seekedRead = false
		cf.seekedEnd = false
		return err
	}

	cf.seekedRead = true
	cf.seekedEnd = false
	return nil
}

// seekEnd seeks the underlying file to the end pointer.
// We reduce sync calls by keeping track of whether we are at the readPtr
// or at the end pointer.
func (cf *circularFile) seekEnd() error {
	if cf.seekedEnd {
		return nil
	}

	_, err := cf.f.Seek(cf.End, io.SeekStart)
	if err != nil {
		cf.seekedRead = false
		cf.seekedEnd = false
		return err
	}

	cf.seekedRead = false
	cf.seekedEnd = true
	return nil
}
