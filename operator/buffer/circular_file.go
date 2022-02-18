package buffer

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var errWriteOverflow = errors.New("bytes to write overflows file")

// circularFile is a io.ReadWriteCloser that writes to a fixed length file, such that
// it wraps around to the beginning when reaching the end.
// Methods on this struct are not thread-safe and require additional synchronization.
type circularFile struct {
	start      int64
	readPtr    int64
	end        int64
	size       int64
	full       bool
	f          fileLike
	seekedRead bool
	seekedEnd  bool
	closed     bool
	readPtrEnd bool
}

var _ io.ReadWriteCloser = (*circularFile)(nil)

func openCircularFile(filePath string, sync bool, size int64, metadata *diskBufferMetadata) (*circularFile, error) {
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
		f.Close()
		return nil, err
	}

	if fsize == 0 {
		err := f.Truncate(size)
		if err != nil {
			f.Close()
			return nil, err
		}
	} else if fsize != size {
		f.Close()
		return nil, fmt.Errorf("configured size (%d) does not match current on-disk size (%d)", size, fsize)
	}

	cf := &circularFile{
		size:    size,
		f:       f,
		start:   metadata.StartOffset,
		readPtr: metadata.StartOffset,
		end:     metadata.EndOffset,
		full:    metadata.Full,
	}

	cf.readPtrEnd = cf.isEmpty()
	return cf, nil
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

func (cf *circularFile) SyncToMetadata(m *diskBufferMetadata) {
	m.StartOffset = cf.start
	m.EndOffset = cf.end
	m.Full = cf.full
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

	var nTotal = 0
	// Read until we've read totalBytesToRead bytes
	for int64(nTotal) < totalBytesToRead {
		bytesToRead := cf.size - cf.readPtr
		if bytesToRead > int64(len(p)-nTotal) {
			bytesToRead = int64(len(p) - nTotal)
		}

		err := cf.seekReadStart()
		if err != nil {
			return nTotal, err
		}

		n, err := cf.f.Read(p[nTotal : int64(nTotal)+bytesToRead])
		nTotal += n

		cf.readPtr = (cf.readPtr + int64(n)) % cf.size
		if cf.readPtr == 0 {
			// We looped back to the start (we read to the end of file),
			// so we are no longer seeked to the read pointer
			cf.seekedRead = false
		}

		if err != nil {
			return nTotal, err
		}
	}

	// If, after a read, our read pointer is at the end of the file, we
	// need to flag that we have emptied the buffer; this differentiates
	// between the "full" case and empty case, since in both cf.ReadPtr == cf.End
	if cf.readPtr == cf.end && totalBytesToRead != 0 {
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

	// Check if we are actually able to fit p in the file.
	// If we cannot, we make only write what we can and return EOF.
	if cf.writeBytesLeft() < int64(len(p)) {
		// p cannot fit into the buffer
		return 0, fmt.Errorf("%w (len(p) == %d, available space == %d)", errWriteOverflow, len(p), cf.writeBytesLeft())
	}

	var nTotal = 0
	// Continue writing until we've written len(p) bytes
	for nTotal < len(p) {
		bytesToWrite := cf.size - cf.end
		if bytesToWrite > int64(len(p)-nTotal) {
			// We can only write up to what's remaining in the buffer!
			bytesToWrite = int64(len(p) - nTotal)
		}

		err = cf.seekEnd()
		if err != nil {
			return nTotal, err
		}

		n, err := cf.f.Write(p[nTotal : int64(nTotal)+bytesToWrite])
		nTotal += n

		if n > 0 {
			// If we wrote anything, the read pointer can no longer be at the end of the buffer.
			cf.readPtrEnd = false
		}

		// Modulus is used here because rb.end may equal rb.size
		cf.end = (cf.end + int64(n)) % cf.size
		if cf.end == 0 {
			// We wrapped around, and are no longer seeked to the end pointer
			cf.seekedEnd = false
		}

		if err != nil {
			return nTotal, err
		}

	}

	if cf.start == cf.end {
		// The buffer is full, so we need to flag this to differentiate from the empty case
		cf.full = true
	}

	return len(p), nil
}

// len gets the current filled size of the buffer.
func (cf *circularFile) len() int64 {
	if cf.full {
		return cf.size
	}

	if cf.start <= cf.end {
		return cf.end - cf.start
	}
	return cf.end + (cf.size - cf.start)
}

// readBytesLeft returns the number of bytes available to read
func (cf *circularFile) readBytesLeft() int64 {
	if cf.isFullyUnread() {
		// The whole buffer is full and unread, in this case
		return cf.size
	}

	if cf.readPtr <= cf.end {
		return cf.end - cf.readPtr
	}

	// Account for wrap around
	return cf.end + (cf.size - cf.readPtr)
}

// writeBytesLeft returns the space in bytes that is available for writing
func (cf *circularFile) writeBytesLeft() int64 {
	return cf.size - cf.len()
}

// isEmpty returns true if there are no bytes currently stored in the buffer
func (cf *circularFile) isEmpty() bool {
	return !cf.full && cf.start == cf.end
}

// isFullyUnread returns true if the buffer is full, and the whole buffer is waiting to be read.
func (cf *circularFile) isFullyUnread() bool {
	return !cf.readPtrEnd && cf.readPtr == cf.end
}

// Discard removes n bytes from the start end of the circular file.
// This resets the internal read pointer to be pointed to start.
// If n is greater than the length of the buffer, the buffer is truncated to a length of 0.
func (cf *circularFile) Discard(n int64) {
	cf.seekedRead = false
	if n <= 0 {
		// If n <= 0, we reset the read pointer anyways for consistency
		cf.readPtr = cf.start
		// If the buffer is empty, set readPtrEnd true
		cf.readPtrEnd = !cf.full && (cf.readPtr == cf.end)
		return
	}

	// discard the bytes by advancing the start pointer,
	// accounting for wraparound
	if n > cf.len() {
		cf.start = cf.end
	} else {
		cf.start = (cf.start + n) % cf.size
	}

	cf.readPtr = cf.start
	cf.full = false
	cf.readPtrEnd = cf.readPtr == cf.end
}

// seekReadStart seeks the underlying file to readPtr.
// We reduce sync calls by keeping track of whether we are at the readPtr
// or at the end pointer.
func (cf *circularFile) seekReadStart() error {
	if cf.seekedRead {
		return nil
	}

	_, err := cf.f.Seek(cf.readPtr, io.SeekStart)
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

	_, err := cf.f.Seek(cf.end, io.SeekStart)
	if err != nil {
		cf.seekedRead = false
		cf.seekedEnd = false
		return err
	}

	cf.seekedRead = false
	cf.seekedEnd = true
	return nil
}
