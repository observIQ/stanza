package file

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/observiq/stanza/errors"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type FileReader struct {
	Path             string
	Fingerprint      Fingerprint
	LastSeenFileSize int64
	Offset           int64

	fileInput *InputOperator

	decoder      *encoding.Decoder
	decodeBuffer []byte

	readInProgress bool

	// This lock must be held any time an exported field
	// on FileReader is written to, or any time it is read from
	// outside the ReadToEnd goroutine
	sync.Mutex         `json:"-"`
	*zap.SugaredLogger `json:"-"`
}

// Initialize sets the starting offset and the initial fingerprint
func (f *FileReader) Initialize(startAtBeginning bool) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}

	buf := make([]byte, 1000)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading fingerprint bytes: %s", err)
	}
	f.Fingerprint.FirstBytes = buf[:n]

	if !startAtBeginning {
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("stat: %s", err)
		}
		f.Offset = info.Size()
	}

	return nil
}

func (f *FileReader) ReadToEnd(ctx context.Context) {
	// Exit early if we are already reading
	if ok := f.setReading(); !ok {
		return
	}
	defer f.unsetReading()

	file, fileSizeHasChanged, err := f.openFile()
	if err != nil {
		f.Errorw("Failed opening file", zap.Error(err))
		return
	}

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

		// TODO if context is cancelled, we don't want to update offset
		f.emit(ctx, scanner.Bytes())
		f.setOffset(scanner.Pos())
	}

	// If we're not at the end of the file, and we haven't
	// advanced since last cycle, read the rest of the file as an entry
	atFileEnd := scanner.Pos() == f.LastSeenFileSize
	if !atFileEnd && fileSizeHasChanged { // TODO why did we have scanner.Pos() == f.offset in here?
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
		f.emit(ctx, msgBuf[:n])
		f.setOffset(scanner.Pos() + int64(n))
	}
}

func (f *FileReader) openFile() (file *os.File, fileSizeHasChanged bool, err error) {
	file, err = os.Open(f.Path)
	if err != nil {
		f.Errorw("Failed to open file", zap.Error(err))
		return nil, false, fmt.Errorf("open file: %s", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("stat file: %s", err)
	}

	f.Lock()
	fileSizeHasChanged = false
	if stat.Size() != f.LastSeenFileSize {
		fileSizeHasChanged = true
		f.LastSeenFileSize = stat.Size()
	}
	if stat.Size() < f.Offset {
		// The file has been truncated, so start from the beginning
		f.Offset = 0
	}
	f.Unlock()

	if _, err = file.Seek(f.Offset, 0); err != nil {
		return nil, false, fmt.Errorf("seek file: %s", err)
	}

	return
}

func (f *FileReader) setOffset(n int64) {
	f.Lock()
	f.Offset = n
	f.Unlock()
}

func (f *FileReader) emit(ctx context.Context, msgBuf []byte) error {
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

// setReading sets readInProgress to true. The return value
// indicates whether readInProgress was changed
func (f *FileReader) setReading() bool {
	f.Lock()
	defer f.Unlock()

	if f.readInProgress {
		return false
	}

	f.readInProgress = true
	return true
}

// unsetReading sets readInProgress to true. The return value
// indicates whether readInProgress was changed
func (f *FileReader) unsetReading() {
	f.Lock()
	defer f.Unlock()
	f.readInProgress = false
}

type Fingerprint struct {
	FirstBytes []byte
}

func (f Fingerprint) Matches(old Fingerprint) bool {
	return bytes.Equal(old.FirstBytes, f.FirstBytes[:len(old.FirstBytes)])
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
