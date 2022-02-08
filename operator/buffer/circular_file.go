package buffer

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/multierr"
)

// CircularFile is a io.ReadWriteCloser that writes to a fixed length file, such that
// it wraps around to the beginning when reaching the end.
// Methods on this struct are not thread-safe and require additional synchronization.
type CircularFile struct {
	Start      int64
	ReadPtr    int64
	End        int64
	Size       int64
	Full       bool
	f          FileLike
	seekedRead bool
	seekedEnd  bool
	closed     bool
	readPtrEnd bool
}

var _ io.ReadWriteCloser = (*CircularFile)(nil)

func OpenCircularFile(filePath string, sync bool, size int64) (*CircularFile, error) {
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

	return &CircularFile{
		Size: size,
		f:    f,
	}, nil
}

func (rb *CircularFile) Close() error {
	if rb.closed {
		return nil
	}

	rb.closed = true

	err := rb.f.Close()
	if err != nil {
		return err
	}

	return nil
}

// Read reads from the ring buffer into p, up to len(p) bytes.
// The contents read are not discarded from the buffer; An independent read pointer
// is maintained, which can be reset to the start of the buffer using ResetReadOffset()
func (rb *CircularFile) Read(p []byte) (int, error) {
	if rb.closed {
		return 0, ErrBufferClosed
	}

	if len(p) == 0 {
		return 0, nil
	}

	err := rb.seekReadStart()
	if err != nil {
		return 0, err
	}

	var totalBytesToRead int64
	var isEOF bool
	if rb.ReadBytesLeft() < int64(len(p)) {
		totalBytesToRead = rb.ReadBytesLeft()
		isEOF = true
	} else {
		totalBytesToRead = int64(len(p))
		isEOF = false
	}

	var firstReadBytes int64
	if totalBytesToRead+rb.ReadPtr <= rb.Size {
		firstReadBytes = totalBytesToRead
	} else {
		firstReadBytes = rb.Size - rb.ReadPtr
	}

	n, err := rb.f.Read(p[:firstReadBytes])
	rb.ReadPtr = (rb.ReadPtr + int64(n)) % rb.Size
	if rb.ReadPtr == 0 {
		rb.seekedRead = false
	}

	if err != nil {
		return n, err
	}

	if firstReadBytes != totalBytesToRead {
		err = rb.seekReadStart()
		if err != nil {
			return n, err
		}

		n2, err := rb.f.Read(p[firstReadBytes:])
		rb.ReadPtr += int64(n2)
		if err != nil {
			return n + n2, err
		}
	}

	if rb.ReadPtr == rb.End && n != 0 {
		rb.readPtrEnd = true
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
func (rb *CircularFile) Write(p []byte) (int, error) {
	if rb.closed {
		return 0, ErrBufferClosed
	}

	if len(p) == 0 {
		return 0, nil
	}

	err := rb.seekEnd()
	if err != nil {
		return 0, err
	}

	var totalBytesToWrite int64
	var isEOF bool
	if rb.WriteBytesLeft() < int64(len(p)) {
		totalBytesToWrite = rb.WriteBytesLeft()
		isEOF = true
	} else {
		totalBytesToWrite = int64(len(p))
		isEOF = false
	}

	var firstWriteBytes int64
	if totalBytesToWrite+rb.End <= rb.Size {
		firstWriteBytes = totalBytesToWrite
	} else {
		firstWriteBytes = rb.Size - rb.End
	}

	n, err := rb.f.Write(p[:firstWriteBytes])
	// Modulus is used here because rb.end may equal rb.size
	rb.End = (rb.End + int64(n)) % rb.Size
	if rb.End == 0 {
		rb.seekedEnd = false
	}

	if n > 0 {
		rb.readPtrEnd = false
	}

	if err != nil {
		return n, err
	}

	if firstWriteBytes != totalBytesToWrite {
		err = rb.seekEnd()
		if err != nil {
			return n, err
		}

		n2, err := rb.f.Write(p[firstWriteBytes:])
		rb.End += int64(n2)

		if n2 > 0 {
			rb.readPtrEnd = false
		}

		if err != nil {
			return n + n2, err
		}
	}

	if isEOF {
		rb.Full = true
		return int(totalBytesToWrite), io.EOF
	}

	if rb.Start == rb.End {
		rb.Full = true
	}

	return int(totalBytesToWrite), nil
}

func (rb *CircularFile) Len() int64 {
	if rb.Full {
		return rb.Size
	}

	if rb.Start <= rb.End {
		return rb.End - rb.Start
	} else {
		return rb.End + (rb.Size - rb.Start)
	}
}

func (rb *CircularFile) ReadBytesLeft() int64 {
	if rb.readPtrEnd {
		return 0
	}

	if rb.Full && rb.ReadPtr == rb.End {
		return rb.Size
	}

	if rb.ReadPtr <= rb.End {
		return rb.End - rb.ReadPtr
	} else {
		return rb.End + (rb.Size - rb.ReadPtr)
	}
}

func (rb *CircularFile) WriteBytesLeft() int64 {
	return rb.Size - rb.Len()
}

// Discard removes n bytes from the start end of the ring buffer.
// This resets the internal read pointer to be pointed to start.
// If n is greater than the length of the buffer, the buffer is truncated to a length of 0.
func (rb *CircularFile) Discard(n int64) {
	rb.seekedRead = false
	if n == 0 {
		rb.ReadPtr = rb.Start
		rb.readPtrEnd = !rb.Full && (rb.ReadPtr == rb.End)
		return
	}

	if n > rb.Len() {
		rb.Start = rb.End
	} else {
		rb.Start = (rb.Start + n) % rb.Size
	}

	rb.ReadPtr = rb.Start
	rb.Full = false
	rb.readPtrEnd = rb.ReadPtr == rb.End
}

// seekReadStart seeks the underlying file to readPtr.
// We reduce sync calls by keeping track of whether we are at the readPtr
// or at the end pointer.
func (rb *CircularFile) seekReadStart() error {
	if rb.seekedRead {
		return nil
	}

	_, err := rb.f.Seek(rb.ReadPtr, io.SeekStart)
	if err != nil {
		rb.seekedRead = false
		rb.seekedEnd = false
		return err
	}

	rb.seekedRead = true
	rb.seekedEnd = false
	return nil
}

// seekEnd seeks the underlying file to the end pointer.
func (rb *CircularFile) seekEnd() error {
	if rb.seekedEnd {
		return nil
	}

	_, err := rb.f.Seek(rb.End, io.SeekStart)
	if err != nil {
		rb.seekedRead = false
		rb.seekedEnd = false
		return err
	}

	rb.seekedRead = false
	rb.seekedEnd = true
	return nil
}
