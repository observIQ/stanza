package file

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

func init() {
	operator.Register("file_input", func() operator.Builder { return NewInputConfig("") })
}

func NewInputConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig:     helper.NewInputConfig(operatorID, "file_input"),
		PollInterval:    operator.Duration{Duration: 200 * time.Millisecond},
		IncludeFileName: true,
		IncludeFilePath: false,
		StartAt:         "end",
		MaxLogSize:      1024 * 1024,
		Encoding:        "nop",
	}
}

// InputConfig is the configuration of a file input operator
type InputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`

	PollInterval    operator.Duration `json:"poll_interval,omitempty"     yaml:"poll_interval,omitempty"`
	Multiline       *MultilineConfig  `json:"multiline,omitempty"         yaml:"multiline,omitempty"`
	IncludeFileName bool              `json:"include_file_name,omitempty" yaml:"include_file_name,omitempty"`
	IncludeFilePath bool              `json:"include_file_path,omitempty" yaml:"include_file_path,omitempty"`
	StartAt         string            `json:"start_at,omitempty"          yaml:"start_at,omitempty"`
	MaxLogSize      int               `json:"max_log_size,omitempty"      yaml:"max_log_size,omitempty"`
	Encoding        string            `json:"encoding,omitempty"          yaml:"encoding,omitempty"`
}

// MultilineConfig is the configuration a multiline operation
type MultilineConfig struct {
	LineStartPattern string `json:"line_start_pattern" yaml:"line_start_pattern"`
	LineEndPattern   string `json:"line_end_pattern"   yaml:"line_end_pattern"`
}

// Build will build a file input operator from the supplied configuration
func (c InputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if len(c.Include) == 0 {
		return nil, fmt.Errorf("required argument `include` is empty")
	}

	// Ensure includes can be parsed as globs
	for _, include := range c.Include {
		_, err := filepath.Match(include, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	// Ensure excludes can be parsed as globs
	for _, exclude := range c.Exclude {
		_, err := filepath.Match(exclude, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	encoding, err := lookupEncoding(c.Encoding)
	if err != nil {
		return nil, err
	}

	splitFunc, err := c.getSplitFunc(encoding)
	if err != nil {
		return nil, err
	}

	var startAtBeginning bool
	switch c.StartAt {
	case "beginning":
		startAtBeginning = true
	case "end":
		startAtBeginning = false
	default:
		return nil, fmt.Errorf("invalid start_at location '%s'", c.StartAt)
	}

	fileNameField := entry.NewNilField()
	if c.IncludeFileName {
		fileNameField = entry.NewLabelField("file_name")
	}

	filePathField := entry.NewNilField()
	if c.IncludeFilePath {
		filePathField = entry.NewLabelField("file_path")
	}

	operator := &InputOperator{
		InputOperator:    inputOperator,
		Include:          c.Include,
		Exclude:          c.Exclude,
		SplitFunc:        splitFunc,
		PollInterval:     c.PollInterval.Raw(),
		persist:          helper.NewScopedDBPersister(context.Database, c.ID()),
		FilePathField:    filePathField,
		FileNameField:    fileNameField,
		runningFiles:     make(map[string]struct{}),
		fileUpdateChan:   make(chan fileUpdateMessage, 10),
		fingerprintBytes: 1000,
		startAtBeginning: startAtBeginning,
		encoding:         encoding,
		MaxLogSize:       c.MaxLogSize,
	}

	return operator, nil
}

var encodingOverrides = map[string]encoding.Encoding{
	"utf-16":   unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf16":    unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf8":     unicode.UTF8,
	"ascii":    unicode.UTF8,
	"us-ascii": unicode.UTF8,
	"nop":      encoding.Nop,
	"":         encoding.Nop,
}

func lookupEncoding(enc string) (encoding.Encoding, error) {
	if encoding, ok := encodingOverrides[strings.ToLower(enc)]; ok {
		return encoding, nil
	}
	encoding, err := ianaindex.IANA.Encoding(enc)
	if err != nil {
		return nil, fmt.Errorf("unsupported encoding '%s'", enc)
	}
	if encoding == nil {
		return nil, fmt.Errorf("no charmap defined for encoding '%s'", enc)
	}
	return encoding, nil
}

// getSplitFunc will return the split function associated the configured mode.
func (c InputConfig) getSplitFunc(encoding encoding.Encoding) (bufio.SplitFunc, error) {
	if c.Multiline == nil {
		return NewNewlineSplitFunc(encoding)
	}
	endPattern := c.Multiline.LineEndPattern
	startPattern := c.Multiline.LineStartPattern

	switch {
	case endPattern != "" && startPattern != "":
		return nil, fmt.Errorf("only one of line_start_pattern or line_end_pattern can be set")
	case endPattern == "" && startPattern == "":
		return nil, fmt.Errorf("one of line_start_pattern or line_end_pattern must be set")
	case endPattern != "":
		re, err := regexp.Compile("(?m)" + c.Multiline.LineEndPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line end regex: %s", err)
		}
		return NewLineEndSplitFunc(re), nil
	case startPattern != "":
		re, err := regexp.Compile("(?m)" + c.Multiline.LineStartPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line start regex: %s", err)
		}
		return NewLineStartSplitFunc(re), nil
	default:
		return nil, fmt.Errorf("unreachable")
	}
}

// InputOperator is an operator that monitors files for entries
type InputOperator struct {
	helper.InputOperator

	Include       []string
	Exclude       []string
	FilePathField entry.Field
	FileNameField entry.Field
	PollInterval  time.Duration
	SplitFunc     bufio.SplitFunc
	MaxLogSize    int

	persist helper.Persister

	runningFiles     map[string]struct{}
	knownFiles       map[string]*knownFileInfo
	startAtBeginning bool

	fileUpdateChan   chan fileUpdateMessage
	fingerprintBytes int64

	encoding encoding.Encoding

	wg       *sync.WaitGroup
	readerWg *sync.WaitGroup
	cancel   context.CancelFunc
}

// Start will start the file monitoring process
func (f *InputOperator) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.wg = &sync.WaitGroup{}
	f.readerWg = &sync.WaitGroup{}

	var err error
	f.knownFiles, err = f.readKnownFiles()
	if err != nil {
		return fmt.Errorf("failed to read known files from database: %s", err)
	}

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer f.syncKnownFiles()
		defer f.drainMessages()

		globTicker := time.NewTicker(f.PollInterval)
		defer globTicker.Stop()

		// All accesses to runningFiles and knownFiles should be done from
		// this goroutine. That means that all private methods of FileInput
		// are unsafe to call from multiple goroutines. Changes to these
		// maps should be done through the fileUpdateChan.
		firstCheck := true
		for {
			select {
			case <-ctx.Done():
				return
			case <-globTicker.C:
				matches := getMatches(f.Include, f.Exclude)
				if firstCheck && len(matches) == 0 {
					f.Warnw("no files match the configured include patterns", "include", f.Include)
				}
				for _, match := range matches {
					f.checkFile(ctx, match, firstCheck)
				}
				f.syncKnownFiles()
				firstCheck = false
			case message, ok := <-f.fileUpdateChan:
				if ok {
					f.updateFile(message)
				}
			}
		}
	}()

	return nil
}

// Stop will stop the file monitoring process
func (f *InputOperator) Stop() error {
	f.cancel()
	f.wg.Wait()
	f.fileUpdateChan = make(chan fileUpdateMessage)
	f.knownFiles = nil
	return nil
}

// checkFile is not safe to call from multiple goroutines
//
// firstCheck indicates whether this is the first time checkFile has been called
// after startup. This is important for the start_at parameter because, after initial
// startup, we don't want to start at the end of newly-created files.
func (f *InputOperator) checkFile(ctx context.Context, path string, firstCheck bool) {

	// Check if the file is currently being read
	if _, ok := f.runningFiles[path]; ok {
		return // file is already being read
	}

	// If the path is known, start from last offset
	knownFile, isKnown := f.knownFiles[path]

	// If the path is new, check if it was from a known file that was rotated
	var err error
	if !isKnown {
		knownFile, err = newKnownFileInfo(path, f.fingerprintBytes, f.startAtBeginning || !firstCheck)
		if err != nil {
			f.Warnw("Failed to get info for file", zap.Error(err))
			return
		}

		for _, knownInfo := range f.knownFiles {
			if knownFile.fingerprintMatches(knownInfo) || knownFile.smallFileContentsMatches(knownInfo) {
				// The file was rotated, so update the path
				knownInfo.Path = path
				knownFile = knownInfo
				break
			}
		}
	}

	f.runningFiles[path] = struct{}{}
	f.knownFiles[path] = knownFile
	f.readerWg.Add(1)
	go func(ctx context.Context, path string, offset, lastSeenSize int64) {
		defer f.readerWg.Done()
		messenger := f.newFileUpdateMessenger(path)
		defer messenger.FinishedReading()
		err := ReadToEnd(ctx, path, offset, lastSeenSize, messenger, f.SplitFunc, f.FilePathField, f.FileNameField, f.InputOperator, f.MaxLogSize, f.encoding)
		if err != nil {
			f.Warnw("Failed to read log file", zap.Error(err))
		}
	}(ctx, path, knownFile.Offset, knownFile.LastSeenFileSize)
}

func (f *InputOperator) updateFile(message fileUpdateMessage) {
	if message.finished {
		delete(f.runningFiles, message.path)
		return
	}

	knownFile := f.knownFiles[message.path]

	// This is a last seen size message, so just set the size and return
	if message.lastSeenFileSize != -1 {
		knownFile.LastSeenFileSize = message.lastSeenFileSize
		return
	}

	if message.newOffset < knownFile.Offset {
		// The file was truncated or rotated

		newKnownFile, err := newKnownFileInfo(message.path, f.fingerprintBytes, true)
		if err != nil {
			f.Warnw("Failed to generate new file info", zap.Error(err))
			return
		}
		f.knownFiles[message.path] = newKnownFile
		return
	}

	if knownFile.Offset < f.fingerprintBytes && message.newOffset > f.fingerprintBytes {
		// The file graduated from small file to fingerprinted file

		file, err := os.Open(message.path)
		if err != nil {
			f.Warnw("Failed to open file for fingerprinting", zap.Error(err))
			return
		}
		defer file.Close()
		knownFile.Fingerprint, err = fingerprintFile(file, f.fingerprintBytes)
		if err != nil {
			f.Warnw("Failed to fingerprint file", zap.Error(err))
			return
		}
		knownFile.IsSmallFile = false
	} else if message.newOffset < f.fingerprintBytes {
		// The file is a small file

		file, err := os.Open(message.path)
		if err != nil {
			f.Warnw("Failed to open small file for content tracking", zap.Error(err))
			return
		}
		defer file.Close()

		buf := make([]byte, message.newOffset)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			f.Warnw("Failed to read small file for content tracking", zap.Error(err))
			return
		}
		knownFile.SmallFileContents = buf[:n]
		knownFile.IsSmallFile = true
	}

	knownFile.Offset = message.newOffset
}

func (f *InputOperator) drainMessages() {
	go func() {
		f.readerWg.Wait()
		close(f.fileUpdateChan)
	}()

	for {
		message, ok := <-f.fileUpdateChan
		if !ok {
			return
		}
		f.updateFile(message)
	}
}

var knownFilesKey = "knownFiles"

func (f *InputOperator) syncKnownFiles() {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(f.knownFiles)
	if err != nil {
		f.Errorw("Failed to encode known files", zap.Error(err))
		return
	}

	f.persist.Set(knownFilesKey, buf.Bytes())
	f.persist.Sync()
}

func (f *InputOperator) readKnownFiles() (map[string]*knownFileInfo, error) {
	err := f.persist.Load()
	if err != nil {
		return nil, err
	}

	var knownFiles map[string]*knownFileInfo
	encoded := f.persist.Get(knownFilesKey)
	if encoded == nil {
		knownFiles = make(map[string]*knownFileInfo)
		return knownFiles, nil
	}

	dec := gob.NewDecoder(bytes.NewReader(encoded))
	err = dec.Decode(&knownFiles)
	if err != nil {
		return nil, err
	}

	return knownFiles, nil
}

func (f *InputOperator) newFileUpdateMessenger(path string) fileUpdateMessenger {
	return fileUpdateMessenger{
		path: path,
		c:    f.fileUpdateChan,
	}
}

type knownFileInfo struct {
	Path              string
	IsSmallFile       bool
	Fingerprint       []byte
	SmallFileContents []byte
	Offset            int64
	LastSeenFileSize  int64
}

func newKnownFileInfo(path string, fingerprintBytes int64, startAtBeginning bool) (*knownFileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var fingerprint []byte
	var smallFileContents []byte
	isSmallFile := false
	size := stat.Size()
	if size > fingerprintBytes {
		fingerprint, err = fingerprintFile(file, fingerprintBytes)
		if err != nil {
			return nil, err
		}
	} else {
		isSmallFile = true
		buf := make([]byte, size)
		n, err := file.Read(buf)
		if err != nil {
			return nil, err
		}
		smallFileContents = buf[:n]
	}

	var offset int64
	if startAtBeginning {
		offset = 0
	} else {
		offset = stat.Size()
	}

	return &knownFileInfo{
		Path:              path,
		Fingerprint:       fingerprint,
		SmallFileContents: smallFileContents,
		IsSmallFile:       isSmallFile,
		Offset:            offset,
	}, nil
}

func (i *knownFileInfo) smallFileContentsMatches(other *knownFileInfo) bool {
	if !(i.IsSmallFile && other.IsSmallFile) {
		return false
	}

	// compare the smaller of the two known files
	var s int
	if len(i.SmallFileContents) > len(other.SmallFileContents) {
		s = len(other.SmallFileContents)
	} else {
		s = len(i.SmallFileContents)
	}

	return bytes.Equal(i.SmallFileContents[:s], other.SmallFileContents[:s])
}

func (i *knownFileInfo) fingerprintMatches(other *knownFileInfo) bool {
	if i.IsSmallFile || other.IsSmallFile {
		return false
	}
	return bytes.Equal(i.Fingerprint, other.Fingerprint)
}

func fingerprintFile(file *os.File, numBytes int64) ([]byte, error) {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	hash := md5.New()

	buffer := make([]byte, numBytes)
	io.ReadFull(file, buffer)
	hash.Write(buffer)
	return hash.Sum(nil), nil
}

type fileUpdateMessage struct {
	path             string
	newOffset        int64
	lastSeenFileSize int64
	finished         bool
}

type fileUpdateMessenger struct {
	c    chan fileUpdateMessage
	path string
}

func (f *fileUpdateMessenger) SetOffset(offset int64) {
	f.c <- fileUpdateMessage{
		path:             f.path,
		newOffset:        offset,
		lastSeenFileSize: -1,
	}
}

func (f *fileUpdateMessenger) SetLastSeenFileSize(size int64) {
	f.c <- fileUpdateMessage{
		path:             f.path,
		lastSeenFileSize: size,
	}
}

func (f *fileUpdateMessenger) FinishedReading() {
	f.c <- fileUpdateMessage{
		path:             f.path,
		finished:         true,
		lastSeenFileSize: -1,
	}
}

func getMatches(includes, excludes []string) []string {
	all := make([]string, 0, len(includes))
	for _, include := range includes {
		matches, _ := filepath.Glob(include) // compile error checked in build
	INCLUDE:
		for _, match := range matches {
			for _, exclude := range excludes {
				if itMatches, _ := filepath.Match(exclude, match); itMatches {
					break INCLUDE
				}
			}

			for _, existing := range all {
				if existing == match {
					break INCLUDE
				}
			}

			all = append(all, match)
		}
	}

	return all
}
