package file

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const defaultFingerprintSize = 1000 // bytes
const minFingerprintSize = 16       // bytes

// Fingerprint is used to identify a file
// A file's fingerprint is the first N bytes of the file,
// where N is the fingerprintSize on the file_input operator
type Fingerprint struct {
	// FirstBytes represents the first N bytes of a file
	FirstBytes []byte
}

// NewFingerprint creates a new fingerprint from an open file
func (f *InputOperator) NewFingerprint(file *os.File) (*Fingerprint, error) {
	buf := make([]byte, f.fingerprintSize)

	n, err := file.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading fingerprint bytes: %s", err)
	}

	fp := &Fingerprint{
		FirstBytes: buf[:n],
	}

	return fp, nil
}

// Copy creates a new copy of the fingerprint
func (f Fingerprint) Copy() *Fingerprint {
	buf := make([]byte, len(f.FirstBytes), cap(f.FirstBytes))
	n := copy(buf, f.FirstBytes)
	return &Fingerprint{
		FirstBytes: buf[:n],
	}
}

// StartsWith returns true if the fingerprints are the same
// or if the new fingerprint starts with the old one
// This is important functionality for tracking new files,
// since their initial size is typically less than that of
// a fingerprint. As the file grows, its fingerprint is updated
// until it reaches a maximum size, as configured on the operator
func (f Fingerprint) StartsWith(old *Fingerprint) bool {
	l0 := len(old.FirstBytes)
	if l0 == 0 {
		return false
	}
	l1 := len(f.FirstBytes)
	if l0 > l1 {
		return false
	}
	return bytes.Equal(old.FirstBytes[:l0], f.FirstBytes[:l0])
}
