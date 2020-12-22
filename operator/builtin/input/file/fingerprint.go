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
type Fingerprint struct {
	FirstBytes []byte
}

// NewFingerprint creates a new fingerprint from an open file
func (f *InputOperator) NewFingerprint(file *os.File) (*Fingerprint, error) {
	return readFingerprint(file, f.fingerprintSize, true)
}

func readFingerprint(file *os.File, len int, retryPartial bool) (*Fingerprint, error) {
	buf := make([]byte, len)
	n, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("reading fingerprint bytes: %s", err)
		}

		/*
			According to file.Read, "At end of file, file.Read returns 0, io.EOF"
			Therefore, we know the file is smaller than the size of the fingerprint
			If we wish to track this file at all, we need to reread with a smaller buffer
		*/
		if n == 0 && retryPartial {
			info, err := file.Stat()
			if err != nil {
				return nil, fmt.Errorf("reading file size: %s", err)
			}
			return readFingerprint(file, int(info.Size()), false)
		}
	}

	return &Fingerprint{
		FirstBytes: buf[:n],
	}, nil
}

// Copy creates a new copy of hte fingerprint
func (f Fingerprint) Copy() *Fingerprint {
	buf := make([]byte, len(f.FirstBytes), cap(f.FirstBytes))
	n := copy(buf, f.FirstBytes)
	return &Fingerprint{
		FirstBytes: buf[:n],
	}
}

// StartsWith returns true if the fingerprints are the same
// or if the new fingerprint starts with the old one
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
