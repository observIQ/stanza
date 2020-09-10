package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	knownFiles       map[string]*Reader
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
	f.knownFiles = nil
	f.currentPollFiles = nil
	f.cancel = nil
	return nil
}

// startPoller kicks off a goroutine that will poll the filesystem periodically,
// checking if there are new files or new logs in the watched files
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

// poll checks all the watched paths for new entries
func (f *InputOperator) poll(ctx context.Context) {
	f.currentPollFiles = make(map[string]*Reader)

	// Get the list of paths on disk
	matches := getMatches(f.Include, f.Exclude)
	if f.firstCheck && len(matches) == 0 {
		f.Warnw("no files match the configured include patterns", "include", f.Include)
	}

	// Populate the currentPollFiles
	// Open the files first to minimize the time between listing and opening
	files := make([]*os.File, 0, len(matches))
	for _, path := range matches {
		file, err := os.Open(path)
		if err != nil {
			f.Errorw("Failed to open file", zap.Error(err))
			continue
		}
		files = append(files, file)
	}

	for _, file := range files {
		if err := f.addFileToCurrent(ctx, file, f.firstCheck); err != nil {
			f.Errorw("Failed to add path", zap.Error(err))
		}
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

	// Wait until all the reader goroutines are finished
	wg.Wait()

	f.rotateCurrent()
	f.syncLastPollFiles()
}

func (f *InputOperator) rotateCurrent() {
	// Rotate current into old
	for path, reader := range f.currentPollFiles {
		reader.file.Close()
		// if old, ok := f.knownFiles[path]; ok {
		// 	// Save the old reader under a random path so we still have the fingerprint
		// 	f.knownFiles[strconv.Itoa(rand.Int())] = old
		// }
		f.knownFiles[path] = reader
	}

	// Clear out old readers
	for path, reader := range f.knownFiles {
		if time.Since(reader.LastSeenTime) > time.Hour {
			delete(f.knownFiles, path)
		}
	}
}

// getMatches gets a list of paths given an array of glob patterns to include and exclude
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

func (f *InputOperator) addFileToCurrent(ctx context.Context, file *os.File, firstCheck bool) error {
	fp, err := NewFingerprint(file)
	if err != nil {
		return fmt.Errorf("create fingerprint: %s", err)
	}

	// Try shortcutting fingerprint check by looking up the old reader by path
	if oldReader, ok := f.knownFiles[file.Name()]; ok {
		if fp.Matches(oldReader.Fingerprint) {
			newReader, err := oldReader.Copy(file)
			if err != nil {
				return err
			}
			f.currentPollFiles[file.Name()] = newReader
			return nil
		}
	}

	// Check if the new path has the same fingerprint as an old path
	for _, oldReader := range f.knownFiles {
		if fp.Matches(oldReader.Fingerprint) {
			// This file has been renamed or copied, so use the offsets from the old reader
			newReader, err := oldReader.Copy(file)
			if err != nil {
				return err
			}
			newReader.Path = file.Name()
			f.currentPollFiles[file.Name()] = newReader
			return nil
		}
	}

	// If we don't match any previously known files, create a new reader from scratch
	newReader, err := NewReader(file.Name(), f, file, fp)
	if err != nil {
		return err
	}
	startAtBeginning := !firstCheck || f.startAtBeginning
	if err := newReader.InitializeOffset(startAtBeginning); err != nil {
		return fmt.Errorf("initialize offset: %s", err)
	}
	f.currentPollFiles[file.Name()] = newReader
	return nil
}

const knownFilesKey = "knownFiles"

// syncLastPollFiles syncs the most recent set of files to the database
func (f *InputOperator) syncLastPollFiles() {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	// Encode the number of known files
	if err := enc.Encode(len(f.knownFiles)); err != nil {
		f.Errorw("Failed to encode known files", zap.Error(err))
		return
	}

	// Encode each known file
	for _, fileReader := range f.knownFiles {
		if err := enc.Encode(fileReader); err != nil {
			f.Errorw("Failed to encode known files", zap.Error(err))
		}
	}

	f.persist.Set(knownFilesKey, buf.Bytes())
	if err := f.persist.Sync(); err != nil {
		f.Errorw("Failed to sync to database", zap.Error(err))
	}
}

// syncLastPollFiles loads the most recent set of files to the database
func (f *InputOperator) loadLastPollFiles() error {
	err := f.persist.Load()
	if err != nil {
		return err
	}

	encoded := f.persist.Get(knownFilesKey)
	if encoded == nil {
		f.knownFiles = make(map[string]*Reader)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(encoded))

	// Decode the number of entries
	var knownFileCount int
	if err := dec.Decode(&knownFileCount); err != nil {
		return fmt.Errorf("decoding file count: %w", err)
	}

	// Decode each of the known files
	f.knownFiles = make(map[string]*Reader)
	for i := 0; i < knownFileCount; i++ {
		newReader, err := NewReader("", f, nil, nil)
		if err != nil {
			return err
		}
		if err = dec.Decode(newReader); err != nil {
			return err
		}
		f.knownFiles[newReader.Path] = newReader
	}

	return nil
}
