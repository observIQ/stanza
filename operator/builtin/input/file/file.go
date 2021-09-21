package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

// InputOperator is an operator that monitors files for entries
type InputOperator struct {
	helper.InputOperator

	finder                Finder
	FilePathField         entry.Field
	FileNameField         entry.Field
	FilePathResolvedField entry.Field
	FileNameResolvedField entry.Field
	PollInterval          time.Duration
	SplitFunc             bufio.SplitFunc
	MaxLogSize            int
	MaxConcurrentFiles    int
	SeenPaths             map[string]time.Time
	filenameRecallPeriod  time.Duration

	persist helper.Persister

	knownFiles      []*Reader
	queuedMatches   []string
	maxBatchFiles   int
	lastPollReaders []*Reader

	startAtBeginning bool
	deleteAfterRead  bool

	fingerprintSize int

	labelRegex *regexp.Regexp

	encoding helper.Encoding

	wg         sync.WaitGroup
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
	for _, reader := range f.lastPollReaders {
		reader.Close()
	}
	f.knownFiles = nil
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
	f.maxBatchFiles = f.MaxConcurrentFiles / 2
	var matches []string
	if len(f.queuedMatches) > f.maxBatchFiles {
		matches, f.queuedMatches = f.queuedMatches[:f.maxBatchFiles], f.queuedMatches[f.maxBatchFiles:]
	} else if len(f.queuedMatches) > 0 {
		matches, f.queuedMatches = f.queuedMatches, make([]string, 0)
	} else {
		// Increment the generation on all known readers
		// This is done here because the next generation is about to start
		for i := 0; i < len(f.knownFiles); i++ {
			f.knownFiles[i].generation++
		}

		// Get the list of paths on disk
		matches = f.finder.FindFiles()
		if f.firstCheck && len(matches) == 0 {
			f.Warnw("no files match the configured include patterns",
				"include", f.finder.Include,
				"exclude", f.finder.Exclude)
		} else if len(matches) > f.maxBatchFiles {
			matches, f.queuedMatches = matches[:f.maxBatchFiles], matches[f.maxBatchFiles:]
		}
	}

	readers := f.makeReaders(ctx, matches)
	f.firstCheck = false

	var wg sync.WaitGroup
	for _, reader := range readers {
		wg.Add(1)
		go func(r *Reader) {
			defer wg.Done()
			r.ReadToEnd(ctx)
		}(reader)
	}
	wg.Wait()

	if f.deleteAfterRead {

		f.Debug("cleaning up log files that have been fully consumed")
		unfinishedReaders := make([]*Reader, 0, len(readers))
		for _, reader := range readers {
			reader.Close()
			if reader.eof {
				if err := os.Remove(reader.file.Name()); err != nil {
					f.Errorf("could not delete %s", reader.file.Name())
				}
			} else {
				unfinishedReaders = append(unfinishedReaders, reader)
			}
		}
		readers = unfinishedReaders

	} else {

		// Detect files that have been rotated out of matching pattern
		lostReaders := make([]*Reader, 0, len(f.lastPollReaders))
	OUTER:
		for _, oldReader := range f.lastPollReaders {
			for _, reader := range readers {
				if reader.Fingerprint.StartsWith(oldReader.Fingerprint) {
					continue OUTER
				}
			}
			lostReaders = append(lostReaders, oldReader)
		}

		for _, reader := range lostReaders {
			wg.Add(1)
			go func(r *Reader) {
				defer wg.Done()
				r.ReadToEnd(ctx)
			}(reader)
		}
		wg.Wait()

		for _, reader := range f.lastPollReaders {
			reader.Close()
		}

		f.lastPollReaders = readers
	}

	f.saveCurrent(readers)
	f.syncLastPollFiles()
}

// makeReaders takes a list of paths, then creates readers from each of those paths,
// discarding any that have a duplicate fingerprint to other files that have already
// been read this polling interval
func (f *InputOperator) makeReaders(ctx context.Context, filePaths []string) []*Reader {
	// Open the files first to minimize the time between listing and opening
	now := time.Now()
	cutoff := now.Add(f.filenameRecallPeriod * -1)
	for filename, lastSeenTime := range f.SeenPaths {
		if lastSeenTime.Before(cutoff) {
			delete(f.SeenPaths, filename)
		}
	}

	files := make([]*os.File, 0, len(filePaths))
	for _, path := range filePaths {
		if _, ok := f.SeenPaths[path]; !ok {
			f.SeenPaths[path] = now
			if f.startAtBeginning {
				f.Infow("Started watching file", "path", path)
			} else {
				f.Infow("Started watching file from end. To read preexisting logs, configure the argument 'start_at' to 'beginning'", "path", path)
			}
		}
		file, err := os.Open(path) // #nosec - operator must read in files defined by user
		if err != nil {
			f.Errorw("Failed to open file", zap.Error(err))
			continue
		}
		files = append(files, file)
	}

	// Get fingerprints for each file
	fps := make([]*Fingerprint, 0, len(files))
	for _, file := range files {
		fp, err := f.NewFingerprint(file)
		if err != nil {
			f.Errorw("Failed creating fingerprint", zap.Error(err))
			continue
		}
		fps = append(fps, fp)
	}

	// Exclude any empty fingerprints or duplicate fingerprints to avoid doubling up on copy-truncate files
OUTER:
	for i := 0; i < len(fps); i++ {
		fp := fps[i]
		if len(fp.FirstBytes) == 0 {
			if err := files[i].Close(); err != nil {
				f.Errorf("problem closing file", "file", files[i].Name())
			}
			// Empty file, don't read it until we can compare its fingerprint
			fps = append(fps[:i], fps[i+1:]...)
			files = append(files[:i], files[i+1:]...)

		}

		for j := i + 1; j < len(fps); j++ {
			fp2 := fps[j]
			if fp.StartsWith(fp2) || fp2.StartsWith(fp) {
				// Exclude
				fps = append(fps[:i], fps[i+1:]...)
				files = append(files[:i], files[i+1:]...)
				continue OUTER
			}
		}
	}

	readers := make([]*Reader, 0, len(fps))
	for i := 0; i < len(fps); i++ {
		reader, err := f.newReader(ctx, files[i], fps[i], f.firstCheck)
		if err != nil {
			f.Errorw("Failed to create reader", zap.Error(err))
			continue
		}
		readers = append(readers, reader)
	}

	return readers
}

// saveCurrent adds the readers from this polling interval to this list of
// known files, then increments the generation of all tracked old readers
// before clearing out readers that have existed for 3 generations.
func (f *InputOperator) saveCurrent(readers []*Reader) {
	// Add readers from the current, completed poll interval to the list of known files
	for _, reader := range readers {
		f.knownFiles = append(f.knownFiles, reader)
	}

	// Clear out old readers. They are sorted such that they are oldest first,
	// so we can just find the first reader whose generation is less than our
	// max, and keep every reader after that
	for i := 0; i < len(f.knownFiles); i++ {
		reader := f.knownFiles[i]
		if reader.generation <= 3 {
			f.knownFiles = f.knownFiles[i:]
			break
		}
	}
}

func (f *InputOperator) newReader(ctx context.Context, file *os.File, fp *Fingerprint, firstCheck bool) (*Reader, error) {
	// Check if the new path has the same fingerprint as an old path
	if oldReader, ok := f.findFingerprintMatch(fp); ok {
		newReader, err := oldReader.Copy(file)
		if err != nil {
			return nil, err
		}
		newReader.fileLabels = f.resolveFileLabels(file.Name())
		return newReader, nil
	}

	// If we don't match any previously known files, create a new reader from scratch
	newReader, err := f.NewReader(file.Name(), file, fp)
	if err != nil {
		return nil, err
	}
	if f.labelRegex != nil {
		/*if err := newReader.readHeaders(ctx); err != nil {
			f.Errorf("error while reading file headers: %s", err)
		}*/
		newReader.ReadHeaders(ctx)
	}
	startAtBeginning := !firstCheck || f.startAtBeginning
	if err := newReader.InitializeOffset(startAtBeginning); err != nil {
		return nil, fmt.Errorf("initialize offset: %s", err)
	}
	return newReader, nil
}

func (f *InputOperator) findFingerprintMatch(fp *Fingerprint) (*Reader, bool) {
	// Iterate backwards to match newest first
	for i := len(f.knownFiles) - 1; i >= 0; i-- {
		oldReader := f.knownFiles[i]
		if fp.StartsWith(oldReader.Fingerprint) {
			return oldReader, true
		}
	}
	return nil, false
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
		f.knownFiles = make([]*Reader, 0, 10)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(encoded))

	// Decode the number of entries
	var knownFileCount int
	if err := dec.Decode(&knownFileCount); err != nil {
		return fmt.Errorf("decoding file count: %w", err)
	}

	// Decode each of the known files
	f.knownFiles = make([]*Reader, 0, knownFileCount)
	for i := 0; i < knownFileCount; i++ {
		newReader, err := f.NewReader("", nil, nil)
		if err != nil {
			return err
		}
		if err = dec.Decode(newReader); err != nil {
			return err
		}
		f.knownFiles = append(f.knownFiles, newReader)
	}

	return nil
}
