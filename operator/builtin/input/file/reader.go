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

var duplicates = sync.Map{}

// Reader manages a single file
type Reader struct {
	Fingerprint      *Fingerprint
	LastSeenFileSize int64
	Offset           int64
	Path             string

	fileInput          *InputOperator
	file               *os.File
	fileSizeHasChanged bool

	decoder      *encoding.Decoder
	decodeBuffer []byte

	*zap.SugaredLogger `json:"-"`
}

// NewReader creates a new file reader
func NewReader(path string, f *InputOperator, file *os.File, fp *Fingerprint) (*Reader, error) {
	r := &Reader{
		Fingerprint:   fp,
		file:          file,
		Path:          path,
		fileInput:     f,
		SugaredLogger: f.SugaredLogger.With("path", path),
		decoder:       f.encoding.NewDecoder(),
		decodeBuffer:  make([]byte, 1<<12),
	}
	if err := r.initialize(); err != nil {
		return nil, err
	}
	return r, nil
}

// Copy creates a deep copy of a Reader
func (f *Reader) Copy(file *os.File) (*Reader, error) {
	buf := make([]byte, 1000)
	n := copy(buf, f.Fingerprint.FirstBytes)
	fp := &Fingerprint{
		FirstBytes: buf[:n],
	}
	// TODO fingerprint.copy()
	reader, err := NewReader(f.Path, f.fileInput, file, fp)
	if err != nil {
		return nil, err
	}

	reader.LastSeenFileSize = f.LastSeenFileSize
	reader.Offset = f.Offset
	if err := reader.initialize(); err != nil {
		return nil, err
	}
	return reader, nil
}

// InitializeOffset sets the starting offset
func (f *Reader) InitializeOffset(startAtBeginning bool) error {
	if !startAtBeginning {
		info, err := f.file.Stat()
		if err != nil {
			return fmt.Errorf("stat: %s", err)
		}
		f.Offset = info.Size()
	}

	return nil
}

func (f *Reader) initialize() error {
	if f.file == nil {
		return nil
	}
	stat, err := f.file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %s", err)
	}

	f.fileSizeHasChanged = false
	if stat.Size() != f.LastSeenFileSize {
		f.fileSizeHasChanged = true
		f.LastSeenFileSize = stat.Size()
	}

	if stat.Size() < f.Offset {
		// The file has been truncated, so start from the beginning
		f.Offset = 0
	}

	return nil
}

// ReadToEnd will read until the end of the file
func (f *Reader) ReadToEnd(ctx context.Context) {
	// lr := io.LimitReader(f.file, f.LastSeenFileSize-f.Offset)
	_, err := f.file.Seek(f.Offset, 0)
	if err != nil {
		f.Errorw("Failed to seek", zap.Error(err))
	}
	fr := NewFingerprintUpdatingReader(f.file, f.Offset, f.Fingerprint)
	scanner := NewPositionalScanner(fr, f.fileInput.MaxLogSize, f.Offset, f.fileInput.SplitFunc)

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
	// atFileEnd := f.Offset == f.LastSeenFileSize
	// if !atFileEnd && !f.fileSizeHasChanged {
	// 	_, err := f.file.Seek(scanner.Pos(), 0)
	// 	if err != nil {
	// 		f.Errorw("Failed to seek for trailing entry", zap.Error(err))
	// 		return
	// 	}

	// 	msgBuf := make([]byte, f.LastSeenFileSize-scanner.Pos())
	// 	n, err := f.file.Read(msgBuf)
	// 	if err != nil {
	// 		f.Errorw("Failed reading trailing entry", zap.Error(err))
	// 		return
	// 	}
	// 	if err := f.emit(ctx, msgBuf[:n]); err != nil {
	// 		f.Error("Failed to emit entry", zap.Error(err))
	// 	}
	// 	f.Offset = scanner.Pos() + int64(n)
	// }
}

func (f *Reader) openFile() (file *os.File, fileSizeHasChanged bool, err error) {
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

	msg := string(f.decodeBuffer[:nDst])
	cp, _ := f.Copy(f.file)
	if previous, ok := duplicates.LoadOrStore(msg, cp); ok {
		fmt.Printf("DUPLICATE: %s\n", msg)
		_ = previous
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

func getScannerError(scanner *PositionalScanner) error {
	err := scanner.Err()
	if err == bufio.ErrTooLong {
		return errors.NewError("log entry too large", "increase max_log_size or ensure that multiline regex patterns terminate")
	} else if err != nil {
		return errors.Wrap(err, "scanner error")
	}
	return nil
}

func NewFingerprintUpdatingReader(r io.Reader, offset int64, f *Fingerprint) *FingerprintUpdatingReader {
	return &FingerprintUpdatingReader{
		fingerprint: f,
		reader:      r,
		offset:      offset,
	}
}

type FingerprintUpdatingReader struct {
	fingerprint *Fingerprint
	reader      io.Reader
	offset      int64
}

func (f *FingerprintUpdatingReader) Read(dst []byte) (int, error) {
	if len(f.fingerprint.FirstBytes) == 1000 {
		return f.reader.Read(dst)
	}
	n, err := f.reader.Read(dst)
	appendCount := min0(n, 1000-int(f.offset))
	f.fingerprint.FirstBytes = append(f.fingerprint.FirstBytes[:f.offset], dst[:appendCount]...)
	f.offset += int64(n)
	return n, err
}

// Fingerprint is used to identify a file
type Fingerprint struct {
	FirstBytes []byte
}

func NewFingerprint(file *os.File) (*Fingerprint, error) {
	buf := make([]byte, 1000)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading fingerprint bytes: %s", err)
	}
	return &Fingerprint{
		FirstBytes: buf[:n],
	}, nil
}

// Matches returns true if the fingerprints are the same
func (f Fingerprint) Matches(old *Fingerprint) bool {
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

func min0(a, b int) int {
	if a < 0 || b < 0 {
		return 0
	}
	if a < b {
		return a
	}
	return b
}
