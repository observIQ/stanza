package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
)

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

	lastPollFiles    map[string]*Reader
	currentPollFiles map[string]*Reader
	startAtBeginning bool

	fingerprintBytes int64

	encoding encoding.Encoding

	wg         sync.WaitGroup
	readerWg   sync.WaitGroup
	firstCheck bool
	cancel     context.CancelFunc
}

// Start will start the file monitoring process
func (f *InputOperator) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.firstCheck = true

	// Load offsets from disk
	if err := f.loadLastPollFiles(); err != nil {
		return fmt.Errorf("read known files from database: %s", err)
	}

	// Start polling goroutine
	f.startPoller(ctx)

	return nil
}

// Stop will stop the file monitoring process
func (f *InputOperator) Stop() error {
	f.cancel()
	f.wg.Wait()
	f.syncLastPollFiles()
	f.lastPollFiles = nil
	f.currentPollFiles = nil
	f.cancel = nil
	return nil
}

// startPoller kicks off a goroutine that will poll the filesystem
// periodically, checking if there are new files or new logs in the
// watched files
func (f *InputOperator) startPoller(ctx context.Context) {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		globTicker := time.NewTicker(f.PollInterval)
		defer globTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-globTicker.C:
			}

			f.poll(ctx)
		}
	}()
}

func (f *InputOperator) poll(ctx context.Context) {
	f.currentPollFiles = make(map[string]*Reader)

	// Get the list of paths on disk
	matches := getMatches(f.Include, f.Exclude)
	if f.firstCheck && len(matches) == 0 {
		f.Warnw("no files match the configured include patterns", "include", f.Include)
	}

	// Populate the currentPollFiles
	for _, match := range matches {
		f.checkPath(ctx, match, f.firstCheck)
	}
	f.firstCheck = false

	// Read all currentPollFiles to end
	var wg sync.WaitGroup
	for _, reader := range f.currentPollFiles {
		wg.Add(1)
		go func(r *Reader) {
			defer wg.Done()
			r.ReadToEnd(ctx)
		}(reader)
	}

	wg.Wait()

	f.lastPollFiles = f.currentPollFiles
	f.syncLastPollFiles()
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

func (f *InputOperator) checkPath(ctx context.Context, path string, firstCheck bool) {
	// Check if we saw this path last time
	if oldReader, ok := f.lastPollFiles[path]; ok {
		f.currentPollFiles[path] = oldReader.Copy()
		return
	}

	// If the path didn't exist last poll, create a new reader
	newReader, err := f.newReader(path, firstCheck)
	if err != nil {
		f.Errorw("Failed to create new reader", zap.Error(err))
		return
	}

	// Check if the new path has the same fingerprint as an old path
	for _, oldReader := range f.lastPollFiles {
		if newReader.Fingerprint.Matches(oldReader.Fingerprint) {
			// This file has been renamed or copied, so use the offsets from the old reader
			newReader.Offset = oldReader.Offset
			newReader.LastSeenFileSize = oldReader.LastSeenFileSize
			break
		}
	}

	f.currentPollFiles[path] = newReader
}

func (f *InputOperator) newReader(path string, firstCheck bool) (*Reader, error) {
	newReader := NewReader(path, f)

	startAtBeginning := !firstCheck || f.startAtBeginning
	if err := newReader.Initialize(startAtBeginning); err != nil {
		return nil, err
	}

	return newReader, nil
}

var knownFilesKey = "knownFiles"

func (f *InputOperator) syncLastPollFiles() {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	// Encode the number of known files
	if err := enc.Encode(len(f.lastPollFiles)); err != nil {
		f.Errorw("Failed to encode known files", zap.Error(err))
		return
	}

	// Encode each known file
	for _, fileReader := range f.lastPollFiles {
		if err := enc.Encode(fileReader); err != nil {
			f.Errorw("Failed to encode known files", zap.Error(err))
		}
	}

	f.persist.Set(knownFilesKey, buf.Bytes())
	if err := f.persist.Sync(); err != nil {
		f.Errorw("Failed to sync to database", zap.Error(err))
	}
}

func (f *InputOperator) loadLastPollFiles() error {
	err := f.persist.Load()
	if err != nil {
		return err
	}

	encoded := f.persist.Get(knownFilesKey)
	if encoded == nil {
		f.lastPollFiles = make(map[string]*Reader)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(encoded))

	// Decode the number of entries
	var knownFileCount int
	if err := dec.Decode(&knownFileCount); err != nil {
		return fmt.Errorf("decoding file count: %w", err)
	}

	// Decode each of the known files
	f.lastPollFiles = make(map[string]*Reader)
	for i := 0; i < knownFileCount; i++ {
		newReader := NewReader("", f)
		if err = dec.Decode(newReader); err != nil {
			return err
		}
		f.lastPollFiles[newReader.Path] = newReader
	}

	return nil
}
