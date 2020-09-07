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

	knownFiles       map[string]*FileReader
	startAtBeginning bool

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

	// Load offsets from disk
	if err := f.loadKnownFiles(); err != nil {
		return fmt.Errorf("read known files from database: %s", err)
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
				return
			case <-globTicker.C:
			}

			f.syncKnownFiles()
			// TODO clean unseen files from our list of known files. This grows unbound
			// if the files rotate
			matches := getMatches(f.Include, f.Exclude)
			if firstCheck && len(matches) == 0 {
				f.Warnw("no files match the configured include patterns", "include", f.Include)
			}
			for _, match := range matches {
				f.checkPath(ctx, match, firstCheck)
			}
			firstCheck = false
		}
	}()

	return nil
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

// Stop will stop the file monitoring process
func (f *InputOperator) Stop() error {
	f.cancel()
	f.wg.Wait()
	f.syncKnownFiles()
	return nil
}

func (f *InputOperator) checkPath(ctx context.Context, path string, firstCheck bool) {
	// Check if we've seen this path before
	reader, ok := f.knownFiles[path]
	if !ok {
		// If we haven't seen it, create a new FileReader
		var err error
		reader, err = f.newFileReader(path, firstCheck)
		if err != nil {
			f.Errorw("Failed to create new reader", zap.Error(err))
			return
		}
		f.knownFiles[path] = reader
	}

	// Read to end
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		reader.ReadToEnd(ctx)
	}()
}

func (f *InputOperator) newFileReader(path string, firstCheck bool) (*FileReader, error) {
	newReader := &FileReader{
		Path:          path,
		fileInput:     f,
		SugaredLogger: f.SugaredLogger.With("path", path),
		decoder:       f.encoding.NewDecoder(),
		decodeBuffer:  make([]byte, 1<<12),
	}

	startAtBeginning := !firstCheck || f.startAtBeginning
	if err := newReader.Initialize(startAtBeginning); err != nil {
		return nil, err
	}

	// Check that this isn't a file we know about that has been moved or rotated
	for oldPath, reader := range f.knownFiles {
		reader.Lock()
		if newReader.Fingerprint.Matches(reader.Fingerprint) {
			// This file has been renamed, so update the path on the
			// old reader and use that instead
			reader.Path = path
			newReader = reader
			delete(f.knownFiles, oldPath)
			reader.Unlock()
			break
		}
		reader.Unlock()
	}

	f.knownFiles[path] = newReader
	return newReader, nil
}

var knownFilesKey = "knownFiles"

func (f *InputOperator) syncKnownFiles() {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(f.knownFiles)
	if err != nil {
		f.Errorw("Failed to encode known files", zap.Error(err))
		return
	}

	f.persist.Set(knownFilesKey, buf.Bytes())
	f.persist.Sync()
}

func (f *InputOperator) loadKnownFiles() error {
	err := f.persist.Load()
	if err != nil {
		return err
	}

	encoded := f.persist.Get(knownFilesKey)
	if encoded == nil {
		f.knownFiles = make(map[string]*FileReader)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(encoded))
	err = dec.Decode(&f.knownFiles)
	if err != nil {
		return err
	}

	for path, knownFile := range f.knownFiles {
		// TODO how to not duplicate this
		knownFile.SugaredLogger = f.SugaredLogger.With("path", path)
		knownFile.fileInput = f
		knownFile.decodeBuffer = make([]byte, 1<<12)
		knownFile.decoder = f.encoding.NewDecoder()
	}

	return nil
}
