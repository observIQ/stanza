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
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("file_input", &FileInputConfig{})
}

type FileInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`

	PollInterval *plugin.Duration           `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
	Multiline    *FileSourceMultilineConfig `json:"multiline,omitempty"     yaml:"multiline,omitempty"`
	PathField    *entry.Field               `json:"path_field,omitempty"    yaml:"path_field,omitempty"`
	StartAt      string                     `json:"start_at,omitempty" yaml:"start_at,omitempty"`
}

type FileSourceMultilineConfig struct {
	LineStartPattern string `json:"line_start_pattern" yaml:"line_start_pattern"`
	LineEndPattern   string `json:"line_end_pattern"   yaml:"line_end_pattern"`
}

func (c FileInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if len(c.Include) == 0 {
		return nil, fmt.Errorf("required argument `include` is empty")
	}

	// Ensure includes can be parsed as globs
	for _, include := range c.Include {
		_, err := filepath.Match(include, "")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	// Ensure excludes can be parsed as globs
	for _, exclude := range c.Exclude {
		_, err := filepath.Match(exclude, "")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	// Determine the split function for log entries
	var splitFunc bufio.SplitFunc
	if c.Multiline == nil {
		splitFunc = bufio.ScanLines
	} else {
		definedLineEndPattern := c.Multiline.LineEndPattern != ""
		definedLineStartPattern := c.Multiline.LineStartPattern != ""

		switch {
		case definedLineEndPattern == definedLineStartPattern:
			return nil, fmt.Errorf("if multiline is configured, exactly one of line_start_pattern or line_end_pattern must be set")
		case definedLineEndPattern:
			re, err := regexp.Compile(c.Multiline.LineEndPattern)
			if err != nil {
				return nil, fmt.Errorf("compile line end regex: %s", err)
			}
			splitFunc = NewLineEndSplitFunc(re)
		case definedLineStartPattern:
			re, err := regexp.Compile(c.Multiline.LineStartPattern)
			if err != nil {
				return nil, fmt.Errorf("compile line start regex: %s", err)
			}
			splitFunc = NewLineStartSplitFunc(re)
		}
	}

	var pollInterval time.Duration
	if c.PollInterval == nil {
		pollInterval = 200 * time.Millisecond
	} else {
		pollInterval = c.PollInterval.Raw()
	}

	var startAtBeginning bool
	switch c.StartAt {
	case "beginning", "":
		startAtBeginning = true
	case "end":
		startAtBeginning = false
	default:
		return nil, fmt.Errorf("invalid start_at location '%s'", c.StartAt)
	}

	plugin := &FileInput{
		InputPlugin:      inputPlugin,
		Include:          c.Include,
		Exclude:          c.Exclude,
		SplitFunc:        splitFunc,
		PollInterval:     pollInterval,
		persist:          helper.NewScopedBBoltPersister(context.Database, c.ID()),
		PathField:        c.PathField,
		runningFiles:     make(map[string]struct{}),
		fileUpdateChan:   make(chan fileUpdateMessage, 10),
		fingerprintBytes: 1000,
		startAtBeginning: startAtBeginning,
	}

	return plugin, nil
}

type FileInput struct {
	helper.InputPlugin

	Include      []string
	Exclude      []string
	PathField    *entry.Field
	PollInterval time.Duration
	SplitFunc    bufio.SplitFunc

	persist helper.Persister

	runningFiles     map[string]struct{}
	knownFiles       map[string]*knownFileInfo
	startAtBeginning bool

	fileUpdateChan   chan fileUpdateMessage
	fingerprintBytes int64

	wg       *sync.WaitGroup
	readerWg *sync.WaitGroup
	cancel   context.CancelFunc
}

func (f *FileInput) Start() error {
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
				f.drainMessages()
				f.readerWg.Wait()
				f.syncKnownFiles()
				return
			case <-globTicker.C:
				matches := getMatches(f.Include, f.Exclude)
				for _, match := range matches {
					f.checkFile(ctx, match, firstCheck)
				}
				f.syncKnownFiles()
				firstCheck = false
			case message := <-f.fileUpdateChan:
				f.updateFile(message)
			}
		}
	}()

	return nil
}

func (f *FileInput) Stop() error {
	f.cancel()
	f.wg.Wait()
	f.syncKnownFiles()
	f.knownFiles = nil
	return nil
}

// checkFile is not safe to call from multiple goroutines
//
// firstCheck indicates whether this is the first time checkFile has been called
// after startup. This is important for the start_at parameter because, after initial
// startup, we don't want to start at the end of newly-created files.
func (f *FileInput) checkFile(ctx context.Context, path string, firstCheck bool) {

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
	go func(ctx context.Context, path string, offset int64) {
		defer f.readerWg.Done()
		messenger := f.newFileUpdateMessenger(path)
		err := ReadToEnd(ctx, path, offset, messenger, f.SplitFunc, f.PathField, f.InputPlugin)
		if err != nil {
			f.Warnw("Failed to read log file", zap.Error(err))
		}
	}(ctx, path, knownFile.Offset)
}

func (f *FileInput) updateFile(message fileUpdateMessage) {
	if message.finished {
		delete(f.runningFiles, message.path)
		return
	}

	knownFile := f.knownFiles[message.path]

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

func (f *FileInput) drainMessages() {
	done := make(chan struct{})
	go func() {
		f.readerWg.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		case message := <-f.fileUpdateChan:
			f.updateFile(message)
		}
	}
}

var knownFilesKey = "knownFiles"

func (f *FileInput) syncKnownFiles() {
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

func (f *FileInput) readKnownFiles() (map[string]*knownFileInfo, error) {
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

func (f *FileInput) newFileUpdateMessenger(path string) fileUpdateMessenger {
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
	path      string
	newOffset int64
	finished  bool
}

type fileUpdateMessenger struct {
	c    chan fileUpdateMessage
	path string
}

func (f *fileUpdateMessenger) SetOffset(offset int64) {
	f.c <- fileUpdateMessage{
		path:      f.path,
		newOffset: offset,
	}
}

func (f *fileUpdateMessenger) FinishedReading() {
	f.c <- fileUpdateMessage{
		path:     f.path,
		finished: true,
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
