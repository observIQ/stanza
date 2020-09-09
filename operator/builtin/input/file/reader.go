package file

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/observiq/stanza/errors"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Reader manages a single file
type Reader struct {
	Fingerprint      Fingerprint
	LastSeenFileSize int64
	Offset           int64
	Path             string

	fileInput *InputOperator

	decoder      *encoding.Decoder
	decodeBuffer []byte

	*zap.SugaredLogger `json:"-"`
}

// NewReader creates a new file reader
func NewReader(path string, f *InputOperator) *Reader {
	return &Reader{
		Path:          path,
		fileInput:     f,
		SugaredLogger: f.SugaredLogger.With("path", path),
		decoder:       f.encoding.NewDecoder(),
		decodeBuffer:  make([]byte, 1<<12),
	}
}

// Copy creates a deep copy of a Reader
func (f *Reader) Copy() *Reader {
	reader := NewReader(f.Path, f.fileInput)
	fingerprint := make([]byte, len(f.Fingerprint.FirstBytes))
	copy(fingerprint, f.Fingerprint.FirstBytes)
	reader.Fingerprint.FirstBytes = fingerprint
	reader.LastSeenFileSize = f.LastSeenFileSize
	reader.Offset = f.Offset
	return reader
}

// Initialize sets the starting offset and the initial fingerprint
func (f *Reader) Initialize(startAtBeginning bool) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := f.Fingerprint.Update(file); err != nil {
		return fmt.Errorf("update fingerprint: %s", err)
	}

	if !startAtBeginning {
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("stat: %s", err)
		}
		f.Offset = info.Size()
	}

	return nil
}

// ReadToEnd will read until the end of the file
func (f *Reader) ReadToEnd(ctx context.Context) {
	file, fileSizeHasChanged, err := f.openFile()
	if err != nil {
		f.Errorw("Failed opening file", zap.Error(err))
		return
	}
	defer file.Close()

	lr := io.LimitReader(file, f.LastSeenFileSize-f.Offset)
	scanner := NewPositionalScanner(lr, f.fileInput.MaxLogSize, f.Offset, f.fileInput.SplitFunc)

	// Iterate over the tokenized file, emitting entries as we go
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ok := scanner.Scan()
		if !ok {
			if err := getScannerError(scanner); err != nil {
				f.Errorw("Failed during scan", zap.Error(err))
			}
			break
		}

		if err := f.emit(ctx, scanner.Bytes()); err != nil {
			f.Error("Failed to emit entry", zap.Error(err))
		}
		f.Offset = scanner.Pos()
	}

	// If we're not at the end of the file, and we haven't
	// advanced since last cycle, read the rest of the file as an entry
	atFileEnd := scanner.Pos() == f.LastSeenFileSize
	if !atFileEnd && !fileSizeHasChanged {
		_, err := file.Seek(scanner.Pos(), 0)
		if err != nil {
			f.Errorw("Failed to seek for trailing entry", zap.Error(err))
			return
		}

		msgBuf := make([]byte, f.LastSeenFileSize-scanner.Pos())
		n, err := file.Read(msgBuf)
		if err != nil {
			f.Errorw("Failed reading trailing entry", zap.Error(err))
			return
		}
		if err := f.emit(ctx, msgBuf[:n]); err != nil {
			f.Error("Failed to emit entry", zap.Error(err))
		}
		f.Offset = scanner.Pos() + int64(n)
	}
}

func (f *Reader) openFile() (file *os.File, fileSizeHasChanged bool, err error) {
	file, err = os.Open(f.Path)
	if err != nil {
		return nil, false, fmt.Errorf("open file: %s", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("stat file: %s", err)
	}

	if stat.Size() != f.LastSeenFileSize {
		fileSizeHasChanged = true
		f.LastSeenFileSize = stat.Size()
	}
	if stat.Size() < f.Offset {
		// The file has been truncated, so start from the beginning
		f.Offset = 0
		if err := f.Fingerprint.Update(file); err != nil {
			return nil, false, fmt.Errorf("update fingerprint: %s", err)
		}
	}

	if _, err = file.Seek(f.Offset, 0); err != nil {
		return nil, false, fmt.Errorf("seek file: %s", err)
	}

	return
}

func (f *Reader) emit(ctx context.Context, msgBuf []byte) error {
	// Skip the entry if it's empty
	if len(msgBuf) == 0 {
		return nil
	}

	f.decoder.Reset()
	var nDst int
	var err error
	for {
		nDst, _, err = f.decoder.Transform(f.decodeBuffer, msgBuf, true)
		if err != nil && err == transform.ErrShortDst {
			f.decodeBuffer = make([]byte, len(f.decodeBuffer)*2)
			continue
		} else if err != nil {
			return fmt.Errorf("transform encoding: %s", err)
		}
		break
	}

	e, err := f.fileInput.NewEntry(string(f.decodeBuffer[:nDst]))
	if err != nil {
		return fmt.Errorf("create entry: %s", err)
	}

	if err := e.Set(f.fileInput.FilePathField, f.Path); err != nil {
		return err
	}
	if err := e.Set(f.fileInput.FileNameField, filepath.Base(f.Path)); err != nil {
		return err
	}
	f.fileInput.Write(ctx, e)
	return nil
}

// Fingerprint is used to identify a file
type Fingerprint struct {
	FirstBytes []byte
}

// Matches returns true if the fingerprints are the same
func (f Fingerprint) Matches(old Fingerprint) bool {
	return bytes.Equal(old.FirstBytes, f.FirstBytes[:len(old.FirstBytes)])
}

func (f *Fingerprint) Update(file *os.File) error {
	buf := make([]byte, 1000)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading fingerprint bytes: %s", err)
	}
	f.FirstBytes = buf[:n]
	return nil
}

func getScannerError(scanner *PositionalScanner) error {
	err := scanner.Err()
	if err == bufio.ErrTooLong {
		return errors.NewError("log entry too large", "increase max_log_size or ensure that multiline regex patterns terminate")
	} else if err != nil {
		return errors.Wrap(err, "scanner error")
	}
	return nil
}
