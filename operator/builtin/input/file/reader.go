package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// File labels contains information about file paths
type fileLabels struct {
	Name         string
	Path         string
	ResolvedName string
	ResolvedPath string
}

// resolveFileLabels resolves file labels
// and sets it to empty string in case of error
func (f *InputOperator) resolveFileLabels(path string) *fileLabels {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		f.Error(err)
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		f.Error(err)
	}

	return &fileLabels{
		Path:         path,
		Name:         filepath.Base(path),
		ResolvedPath: abs,
		ResolvedName: filepath.Base(abs),
	}
}

// Reader manages a single file
type Reader struct {
	Fingerprint *Fingerprint
	Offset      int64

	// HeaderLabels is an optional map that contains entry labels
	// derived from a log files' headers, added to every record
	HeaderLabels map[string]string

	generation int
	fileInput  *InputOperator
	file       *os.File
	fileLabels *fileLabels

	decoder      *encoding.Decoder
	decodeBuffer []byte

	*zap.SugaredLogger `json:"-"`
}

// NewReader creates a new file reader
func (f *InputOperator) NewReader(path string, file *os.File, fp *Fingerprint) (*Reader, error) {
	r := &Reader{
		Fingerprint:   fp,
		HeaderLabels:  make(map[string]string),
		file:          file,
		fileInput:     f,
		SugaredLogger: f.SugaredLogger.With("path", path),
		decoder:       f.encoding.Encoding.NewDecoder(),
		decodeBuffer:  make([]byte, 1<<12),
		fileLabels:    f.resolveFileLabels(path),
	}
	return r, nil
}

// Copy creates a deep copy of a Reader
func (f *Reader) Copy(file *os.File) (*Reader, error) {
	reader, err := f.fileInput.NewReader(f.fileLabels.Path, file, f.Fingerprint.Copy())
	if err != nil {
		return nil, err
	}
	reader.Offset = f.Offset
	for k, v := range f.HeaderLabels {
		reader.HeaderLabels[k] = v
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

type consumerFunc func(context.Context, []byte) error

// ReadToEnd will read until the end of the file
func (f *Reader) ReadToEnd(ctx context.Context) {
	f.readFile(ctx, f.emit)
}

// ReadHeaders will read a files headers
func (f *Reader) ReadHeaders(ctx context.Context) {
	f.readFile(ctx, f.readHeaders)
}

func (f *Reader) readFile(ctx context.Context, consumer consumerFunc) {
	if _, err := f.file.Seek(f.Offset, 0); err != nil {
		f.Errorw("Failed to seek", zap.Error(err))
		return
	}
	fr := NewFingerprintUpdatingReader(f.file, f.Offset, f.Fingerprint, f.fileInput.fingerprintSize)
	scanner := NewPositionalScanner(fr, f.fileInput.MaxLogSize, f.Offset, f.fileInput.SplitFunc)

	// Iterate over the tokenized file
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
		if err := consumer(ctx, scanner.Bytes()); err != nil {
			// return if header parsing is done
			if err == errEndOfHeaders {
				return
			}
			f.Error("Failed to consume entry", zap.Error(err))
		}
		f.Offset = scanner.Pos()
	}
}

var errEndOfHeaders = fmt.Errorf("finished header parsing, no header found")

func (f *Reader) readHeaders(ctx context.Context, msgBuf []byte) error {
	byteMatches := f.fileInput.labelRegex.FindSubmatch(msgBuf)
	if len(byteMatches) != 3 {
		// return early, assume this failure means the file does not
		// contain anymore headers
		return errEndOfHeaders
	}
	matches := make([]string, len(byteMatches))
	for i, byteSlice := range byteMatches {
		matches[i] = string(byteSlice)
	}
	if f.HeaderLabels == nil {
		f.HeaderLabels = make(map[string]string)
	}
	f.HeaderLabels[matches[1]] = matches[2]
	return nil
}

// Close will close the file
func (f *Reader) Close() {
	if f.file != nil {
		if err := f.file.Close(); err != nil {
			f.Debugf("Problem closing reader", "Error", err.Error())
		}
	}
}

// Emit creates an entry with the decoded message and sends it to the next
// operator in the pipeline
func (f *Reader) emit(ctx context.Context, msgBuf []byte) error {
	// Skip the entry if it's empty
	if len(msgBuf) == 0 {
		return nil
	}

	msg, err := f.decode(msgBuf)
	if err != nil {
		return fmt.Errorf("decode: %s", err)
	}

	e, err := f.fileInput.NewEntry(msg)
	if err != nil {
		return fmt.Errorf("create entry: %s", err)
	}

	if err := e.Set(f.fileInput.FilePathField, f.fileLabels.Path); err != nil {
		return err
	}
	if err := e.Set(f.fileInput.FileNameField, filepath.Base(f.fileLabels.Path)); err != nil {
		return err
	}

	if err := e.Set(f.fileInput.FilePathResolvedField, f.fileLabels.ResolvedPath); err != nil {
		return err
	}
	if err := e.Set(f.fileInput.FileNameResolvedField, f.fileLabels.ResolvedName); err != nil {
		return err
	}

	// Set W3C headers as labels
	for k, v := range f.HeaderLabels {
		field := entry.NewLabelField(k)
		if err := e.Set(field, v); err != nil {
			return err
		}
	}

	f.fileInput.Write(ctx, e)
	return nil
}

// decode converts the bytes in msgBuf to utf-8 from the configured encoding
func (f *Reader) decode(msgBuf []byte) (string, error) {
	for {
		f.decoder.Reset()
		nDst, _, err := f.decoder.Transform(f.decodeBuffer, msgBuf, true)
		if err != nil && err == transform.ErrShortDst {
			f.decodeBuffer = make([]byte, len(f.decodeBuffer)*2)
			continue
		} else if err != nil {
			return "", fmt.Errorf("transform encoding: %s", err)
		}
		return string(f.decodeBuffer[:nDst]), nil
	}
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

// NewFingerprintUpdatingReader creates a new FingerprintUpdatingReader starting starting at the given offset
func NewFingerprintUpdatingReader(r io.Reader, offset int64, f *Fingerprint, fingerprintSize int) *FingerprintUpdatingReader {
	return &FingerprintUpdatingReader{
		fingerprint:     f,
		fingerprintSize: fingerprintSize,
		reader:          r,
		offset:          offset,
	}
}

// FingerprintUpdatingReader wraps another reader, and updates the fingerprint
// with each read in the first fingerPrintSize bytes
type FingerprintUpdatingReader struct {
	fingerprint     *Fingerprint
	fingerprintSize int
	reader          io.Reader
	offset          int64
}

// Read reads from the wrapped reader, saving the read bytes to the fingerprint
func (f *FingerprintUpdatingReader) Read(dst []byte) (int, error) {
	if len(f.fingerprint.FirstBytes) == f.fingerprintSize {
		return f.reader.Read(dst)
	}
	n, err := f.reader.Read(dst)
	appendCount := min0(n, f.fingerprintSize-int(f.offset))
	f.fingerprint.FirstBytes = append(f.fingerprint.FirstBytes[:f.offset], dst[:appendCount]...)
	f.offset += int64(n)
	return n, err
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
