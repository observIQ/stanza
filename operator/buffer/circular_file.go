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
		Size:       size,
		f:          f,
		readPtrEnd: true,
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
// is maintained.
// The "discard" method may be used to remove bytes from the buffer, as well as to reset the read pointer.
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
	// Check if the amount of bytes left for us to read can fill the buffer
	if cf.readBytesLeft() < int64(len(p)) {
		// we cannot fill the whole buffer, we will return EOF.
		totalBytesToRead = cf.readBytesLeft()
		isEOF = true
	} else {
		// we can fill the whole buffer
		totalBytesToRead = int64(len(p))
		isEOF = false
	}

	var firstReadBytes int64
	// Check if out read will cross the end of the flat file on disk;
	// if it does, we need to split this read into 2 actual reads of the underlying file.
	if totalBytesToRead+cf.ReadPtr <= cf.Size {
		// We do not cross the boundary, we can fill the whole buffer in one read
		firstReadBytes = totalBytesToRead
	} else {
		// Need to split into 2 reads, one to the end, and from the start.
		firstReadBytes = cf.Size - cf.ReadPtr
	}

	n, err := cf.f.Read(p[:firstReadBytes])
	cf.ReadPtr = (cf.ReadPtr + int64(n)) % cf.Size
	if cf.ReadPtr == 0 {
		// We looped back to the start (we read to the end of file),
		// so we are no longer seeked to the read pointer
		cf.seekedRead = false
	}

	if err != nil {
		return n, err
	}

	// We haven't read the full amount we can read, meaning we need to do a second read
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

	// If, after a read, our read pointer is at the end of the file, we
	// need to flag that we have emptied the buffer; this differentiates
	// between the "full" case and empty case, since in both cf.ReadPtr == cf.End
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
	// Check if we are actually able to fit p in the file.
	// If we cannot, we make only write what we can and return EOF.
	if cf.writeBytesLeft() < int64(len(p)) {
		// Can only partially write p
		totalBytesToWrite = cf.writeBytesLeft()
		isEOF = true
	} else {
		// Whole buffer can fit on disk
		totalBytesToWrite = int64(len(p))
		isEOF = false
	}

	var firstWriteBytes int64
	if cf.End+totalBytesToWrite <= cf.Size {
		// We can write the whole buffer contiguously
		firstWriteBytes = totalBytesToWrite
	} else {
		// We need to wrap around, and lay down the buffer in 2 writes.
		firstWriteBytes = cf.Size - cf.End
	}

	n, err := cf.f.Write(p[:firstWriteBytes])
	// Modulus is used here because rb.end may equal rb.size
	cf.End = (cf.End + int64(n)) % cf.Size
	if cf.End == 0 {
		// We wrapped around, and are no longer seeked to the end pointer
		cf.seekedEnd = false
	}

	if n > 0 {
		// If we wrote any bytes, the read pointer would no longer be at the end pointer, since the end has advanced.
		cf.readPtrEnd = false
	}

	if err != nil {
		return n, err
	}

	if firstWriteBytes != totalBytesToWrite {
		// We didn't write the whole buffer in the first go, due to wraparound,
		// so we need a second write
		err = cf.seekEnd()
		if err != nil {
			return n, err
		}

		n2, err := cf.f.Write(p[firstWriteBytes:])
		cf.End += int64(n2)

		if n2 > 0 {
			// If we wrote any bytes, the read pointer would no longer be at the end pointer, since the end has advanced.
			cf.readPtrEnd = false
		}

		if err != nil {
			return n + n2, err
		}
	}

	if isEOF {
		// If we hit EOF writing, this means the buffer is full
		cf.Full = true
		return int(totalBytesToWrite), io.EOF
	}

	if cf.Start == cf.End && totalBytesToWrite != 0 {
		// The buffer is full, so we need to flag this to differentiate from the empty case
		cf.Full = true
	}

	return int(totalBytesToWrite), nil
}

// len gets the current filled size of the buffer.
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

// readBytesLeft returns the number of bytes available to read
func (cf *circularFile) readBytesLeft() int64 {
	if !cf.readPtrEnd && cf.ReadPtr == cf.End {
		// The whole buffer is full and unread, in this case
		return cf.Size
	}

	if cf.ReadPtr <= cf.End {
		return cf.End - cf.ReadPtr
	} else {
		// Account for wrap around
		return cf.End + (cf.Size - cf.ReadPtr)
	}
}

// writeBytesLeft returns the space in bytes that is available for writing
func (cf *circularFile) writeBytesLeft() int64 {
	return cf.Size - cf.len()
}

// discard removes n bytes from the start end of the circular file.
// This resets the internal read pointer to be pointed to start.
// If n is greater than the length of the buffer, the buffer is truncated to a length of 0.
func (cf *circularFile) discard(n int64) {
	cf.seekedRead = false
	if n <= 0 {
		// If n <= 0, we reset the read pointer anyways for consistency
		cf.ReadPtr = cf.Start
		cf.seekedRead = false
		// If the buffer is empty, set readPtrEnd true
		cf.readPtrEnd = !cf.Full && (cf.ReadPtr == cf.End)
		return
	}

	// discard the bytes by advancing the start pointer,
	// accounting for wraparound
	if n > cf.len() {
		cf.Start = cf.End
	} else {
		cf.Start = (cf.Start + n) % cf.Size
	}

	cf.ReadPtr = cf.Start
	cf.seekedRead = false
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
