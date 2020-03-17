package fileinput

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("file", &FileSourceConfig{})
}

type FileSourceConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`

	Include []string `yaml:",omitempty"`
	Exclude []string `yaml:",omitempty"`
	// TODO start from beginning once offsets are implemented
	PollInterval float64                    `yaml:",omitempty"`
	Multiline    *FileSourceMultilineConfig `yaml:",omitempty"`
}

type FileSourceMultilineConfig struct {
	LineStartPattern string `mapstructure:"line_start_pattern" yaml:"line_start_pattern"`
	LineEndPattern   string `mapstructure:"line_end_pattern" yaml:"line_end_pattern"`
}

func (c FileSourceConfig) Build(buildContext pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(buildContext.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(buildContext.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
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

	// Parse the poll interval
	if c.PollInterval < 0 {
		return nil, fmt.Errorf("poll_interval must be greater than zero if configured")
	}
	pollInterval := func() time.Duration {
		if c.PollInterval == 0 {
			return 5 * time.Second
		} else {
			return time.Duration(float64(time.Second) * c.PollInterval)
		}
	}()

	plugin := &FileSource{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,

		Include:          c.Include,
		Exclude:          c.Exclude,
		SplitFunc:        splitFunc,
		PollInterval:     pollInterval,
		FingerprintBytes: 100,

		fileCreated: make(chan string),
		offsetStore: &OffsetStore{
			db:     buildContext.Database,
			bucket: string(c.ID()), // TODO use bundle as prefix
		},
	}

	return plugin, nil
}

type FileSource struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	Include          []string
	Exclude          []string
	PollInterval     time.Duration
	SplitFunc        bufio.SplitFunc
	FingerprintBytes int64

	wg     *sync.WaitGroup
	cancel context.CancelFunc

	fileWatchers      []*FileWatcher
	fileMux           sync.Mutex
	directoryWatchers map[string]*DirectoryWatcher
	directoryMux      sync.Mutex

	fileCreated chan string

	offsetStore *OffsetStore
}

func (f *FileSource) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.wg = &sync.WaitGroup{}

	f.fileWatchers = make([]*FileWatcher, 0)
	f.directoryWatchers = make(map[string]*DirectoryWatcher, 0)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer f.Info("Exiting glob updater")

		// Do it once first for responsive startup
		matches := globMatches(f.Include, f.Exclude)
		for _, match := range matches {
			f.tryAddFile(ctx, match, true)
		}

		// TODO clear database of untracked offsets after initial startup

		globTicker := time.NewTicker(f.PollInterval)
		defer globTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-globTicker.C:
				matches := globMatches(f.Include, f.Exclude)
				for _, match := range matches {
					f.tryAddFile(ctx, match, true)
				}
			case path := <-f.fileCreated:
				f.Debugw("Received file created notification", "path", path)
				f.tryAddFile(ctx, path, false)
			}
		}
	}()

	return nil
}

func (f *FileSource) Stop() {
	f.Info("Stopping source")
	f.cancel()
	f.wg.Wait()
	f.Info("Stopped source")
}

// globMatches queries the filesystem for any files that match one of the
// include patterns, but do not match any of the exclude patterns
func globMatches(includes []string, excludes []string) []string {
	matched := []string{}
	for _, includePattern := range includes {
		matches, _ := filepath.Glob(includePattern)
		for _, path := range matches {
			fileInfo, err := os.Stat(path)
			if err != nil || fileInfo.IsDir() {
				continue // skip directories
			}
			if isExcluded(path, excludes) {
				continue // skip excluded
			}
			matched = append(matched, path)
		}
	}
	return matched
}

// isExcluded checks if the path is matched by any of the exclude patterns
func isExcluded(path string, excludes []string) bool {
	for _, excludePattern := range excludes {
		// error already checked in build step
		if exclude, _ := filepath.Match(excludePattern, path); exclude {
			return true
		}
	}

	return false
}

//
func (f *FileSource) tryAddFile(ctx context.Context, path string, globCheck bool) {
	// Skip the path if it's excluded
	if isExcluded(path, f.Exclude) {
		f.Debugw("Skipping excluded file", "path", path)
		return
	}

	// Add the file's directory so we can get faster notifications
	f.tryAddDirectory(ctx, filepath.Dir(path))

	// Check if we should start watching the file
	createWatcher, startingOffset, err := f.checkPath(path, !globCheck)
	if err != nil || !createWatcher {
		return
	}

	// Create the file watcher
	watcher := &FileWatcher{
		path:             path,
		offset:           startingOffset,
		pollInterval:     f.PollInterval,
		splitFunc:        f.SplitFunc,
		output:           f.Output,
		fingerprintBytes: f.FingerprintBytes,
		offsetStore:      f.offsetStore,
		SugaredLogger:    f.SugaredLogger.With("path", path),
	}

	// Save a reference
	fp, _ := watcher.Fingerprint()
	f.Infow("Watching file", "path", watcher.path, "offset", watcher.offset, "fingerprint", fp)
	f.overwriteFileWatcher(watcher)

	// Start the watcher
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer f.Debugw("File watcher stopped", "path", path)
		defer f.removeFileWatcher(watcher)

		err := watcher.Watch(ctx)
		if err != nil {
			if pathError, ok := err.(*os.PathError); ok && pathError.Err.Error() == "no such file or directory" {
				f.Debugw("File deleted before it could be read", "path", path)
			} else {
				f.Warnw("Watch failed", "error", err)
			}
		}
	}()
}

func (f *FileSource) checkPath(path string, checkCopy bool) (createWatcher bool, startingOffset int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return false, 0, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return false, 0, err
	}

	// If zero size, skip fingerprinting
	if fileInfo.Size() == 0 {
		for _, watcher := range f.fileWatchers {
			if watcher.path == path {
				return false, 0, nil
			}
		}
		return true, 0, nil
	}

	fingerprint, _ := fingerprint(f.FingerprintBytes, file)

	// Skip if fingerprint and path are the same
	for _, watcher := range f.fileWatchers {
		if watcher.path == path {
			watcherFP, err := watcher.Fingerprint()
			if err != nil {
				f.Debug("Failed to get watcher fingerprint", "error", err)
				continue
			}

			if string(watcherFP) == string(fingerprint) {
				return false, 0, nil
			}
		}
	}

	// Detect file rotation (offset is stored but path is different)
	offset, err := f.offsetStore.GetOffset(fingerprint)
	if err != nil {
		f.Warnw("Failed to get offset for fingerprint", "error", err)
	}
	if offset != nil {
		return true, *offset, nil
	}

	return true, 0, nil
}

func fingerprint(numBytes int64, file *os.File) (fp []byte, stable bool) {
	// TODO make sure resetting the seek location isn't messing with things
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		panic(err)
	}
	hash := md5.New()

	buffer := make([]byte, numBytes)
	bytesRead, _ := io.ReadFull(file, buffer)
	// TODO what if the file is empty?
	hash.Write(buffer)
	return hash.Sum(nil), int64(bytesRead) >= numBytes
}

func (f *FileSource) tryAddDirectory(ctx context.Context, path string) {

	_, ok := f.directoryWatchers[path]
	if ok {
		return
	}

	watcher, err := NewDirectoryWatcher(path, f)
	if err != nil {
		f.Warnw("Failed to create directory watcher", "error", err)
		return
	}

	f.directoryWatchers[path] = watcher
	f.Infow("Watching directory", "path", path)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer f.Debugw("Directory watcher stopped", "path", path)
		defer f.removeDirectoryWatcher(watcher)

		err := watcher.Watch(ctx)
		if err != nil {
			f.Warnw("Directory watch failed", "error", err)
		}
	}()
}

func (f *FileSource) removeDirectoryWatcher(directoryWatcher *DirectoryWatcher) {
	f.directoryMux.Lock()
	delete(f.directoryWatchers, directoryWatcher.path)
	f.directoryMux.Unlock()
}

func (f *FileSource) removeFileWatcher(watcher *FileWatcher) {
	f.fileMux.Lock()
	for i, trackedWatcher := range f.fileWatchers {
		if trackedWatcher == watcher {
			trackedWatcher.Exit()
			f.fileWatchers = append(f.fileWatchers[:i], f.fileWatchers[i+1:]...)
		}
	}
	f.fileMux.Unlock()
}

func (f *FileSource) overwriteFileWatcher(watcher *FileWatcher) {
	f.fileMux.Lock()
	overwritten := false
	for i, trackedWatcher := range f.fileWatchers {
		if trackedWatcher.path == watcher.path {
			trackedWatcher.Exit()
			f.fileWatchers[i] = watcher
			overwritten = true
		}
	}

	if !overwritten {
		f.fileWatchers = append(f.fileWatchers, watcher)
	}
	f.fileMux.Unlock()
}
