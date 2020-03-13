package fileinput

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
	// using Docker's filenotify so we can fall back to polling
	// for envs where notify isn't available
)

func init() {
	pg.RegisterConfig("file", &FileSourceConfig{})
}

type FileSourceConfig struct {
	pg.DefaultPluginConfig
	pg.DefaultOutputterConfig

	Include              []string
	Exclude              []string
	FallbackPollInterval float64
	Multiline            *FileSourceMultilineConfig
}

type FileSourceMultilineConfig struct {
	LineStartPattern string `mapstructure:"log_start_pattern"`
	LineEndPattern   string `mapstructure:"log_end_pattern"`
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

	for _, include := range c.Include {
		_, err := filepath.Match(include, "")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	for _, exclude := range c.Exclude {
		_, err := filepath.Match(exclude, "")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	var splitFunc bufio.SplitFunc
	if c.Multiline == nil {
		splitFunc = bufio.ScanLines
	} else if c.Multiline.LineEndPattern != "" && c.Multiline.LineStartPattern != "" {
		return nil, fmt.Errorf("cannot configure both line_start_pattern and line_end_pattern")
	} else if c.Multiline.LineEndPattern != "" {
		re, err := regexp.Compile(c.Multiline.LineEndPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line end regex: %s", err)
		}
		splitFunc = NewLineEndSplitFunc(re)
	} else if c.Multiline.LineStartPattern != "" {
		re, err := regexp.Compile(c.Multiline.LineStartPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line start regex: %s", err)
		}
		splitFunc = NewLineStartSplitFunc(re)
	} else if c.Multiline.LineEndPattern == "" && c.Multiline.LineStartPattern == "" {
		return nil, fmt.Errorf("if multiline is configured, either line_start_pattern or line_end_pattern must be configured")
	}

	plugin := &FileSource{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,

		Include:   c.Include,
		Exclude:   c.Exclude,
		splitFunc: splitFunc,

		fileCreated:      make(chan string),
		fileTouched:      make(chan string),
		fileRemoved:      make(chan *FileWatcher),
		directoryRemoved: make(chan *DirectoryWatcher),
	}

	return plugin, nil
}

type FileSource struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	Include   []string
	Exclude   []string
	splitFunc bufio.SplitFunc

	fingerprintBytes int64

	wg     *sync.WaitGroup
	cancel context.CancelFunc

	fileWatchers      []*FileWatcher
	fileMux           sync.Mutex
	directoryWatchers map[string]*DirectoryWatcher
	directoryMux      sync.Mutex

	fileCreated      chan string
	fileRemoved      chan *FileWatcher
	fileTouched      chan string
	directoryRemoved chan *DirectoryWatcher
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

		// Do it once first
		f.checkGlob(ctx)

		globTicker := time.NewTicker(time.Second) // TODO tune this param and make it configurable

		// Synchronize all new tracking notifications here so there
		// are no race conditions in file operations.
		// Also keeps us from having to do lots of map locking
		for {
			select {
			case <-ctx.Done():
				return
			case <-globTicker.C:
				f.checkGlob(ctx)
			case path := <-f.fileCreated:
				f.Debugw("Received file created notification", "path", path)
				f.tryAddFile(ctx, path, false)
			case watcher := <-f.fileRemoved:
				f.Debugw("Received file removed notification", "path", watcher.path)
				f.removeFileWatcher(watcher)
			case watcher := <-f.directoryRemoved:
				f.Debugw("Received directory removed notification", "path", watcher.path)
				f.removeDirectoryWatcher(watcher)
			case path := <-f.fileTouched:
				f.Debugw("Received file touched notification", "path", path)
				// Don't do anything here, but it ensures that we're not
				// reading at the same time that we're handling file
				// changes
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

func (f *FileSource) checkGlob(ctx context.Context) {
	for _, includePattern := range f.Include {
		matches, _ := filepath.Glob(includePattern)
		for _, path := range matches {
			fileInfo, err := os.Stat(path)
			if err != nil || fileInfo.IsDir() {
				continue // skip directories
			}
			f.tryAddFile(ctx, path, true)
		}
	}
}

func (f *FileSource) isExcluded(path string) bool {
	for _, excludePattern := range f.Exclude {
		// error already checked in build step
		if exclude, _ := filepath.Match(excludePattern, path); exclude {
			return true
		}
	}

	return false
}

func (f *FileSource) tryAddFile(ctx context.Context, path string, globCheck bool) {
	if f.isExcluded(path) {
		f.Debugw("Skipping excluded file", "path", path)
		return
	}

	f.tryAddDirectory(ctx, filepath.Dir(path))

	createWatcher, startFromBeginning, err := f.checkPath(path, !globCheck)
	if !createWatcher {
		return
	}

	watcher, err := NewFileWatcher(path, f, startFromBeginning, f.splitFunc)
	if err != nil {
		if pathError, ok := err.(*os.PathError); ok && pathError.Err.Error() == "no such file or directory" {
			f.Debugw("File deleted before it could be read", "path", path)
		} else {
			f.Warnw("Failed to create file watcher", "error", err)

			fmt.Printf("ERROR TYPE: %T", err)
		}
		return
	}

	f.Infow("Watching file", "path", watcher.path)
	f.overwriteFileWatcher(watcher)

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

func (f *FileSource) checkPath(path string, checkCopy bool) (createWatcher bool, startFromBeginning bool, err error) {
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return false, false, err
	}

	// TODO get these safely
	var inode uint64
	var dev uint64
	switch sys := fileInfo.Sys().(type) {
	case *syscall.Stat_t:
		inode = sys.Ino
		dev = uint64(sys.Dev)
	default:
		return false, false, fmt.Errorf("cannot use fileinfo of type %T", fileInfo.Sys())
	}

	for _, watcher := range f.fileWatchers {
		// TODO what if multiple match? anything?
		if watcher.dev == dev && watcher.inode == inode {
			if watcher.path == path {
				f.Infow("Continuing to watch file", "path", path)
				return false, false, nil
			} else {
				f.Infow("File was renamed", "path", path)
				watcher.path = path
				return false, false, nil
			}
			// Don't check fingerprints during glob check because we only want to
			// check newly-created files. TODO make this cleaner/clearer
		} else if checkCopy && f.fingerprint(watcher.file) == f.fingerprint(file) {
			f.Infow("File was copied. Starting from previous offset", "path", path)
			return true, false, nil
		}
	}

	return true, true, nil
}

func (f *FileSource) fingerprint(file *os.File) string {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		panic(err)
	}
	hash := md5.New()

	buffer := make([]byte, f.fingerprintBytes)
	_, err = io.ReadFull(file, buffer)
	if err != nil {
		panic(err) // TODO
	}
	hash.Write(buffer)
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (f *FileSource) tryAddDirectory(ctx context.Context, path string) {

	_, ok := f.directoryWatchers[path]
	if ok {
		return
	}

	watcher, err := NewDirectoryWatcher(path, f)
	if err != nil {
		println("Creating directory watcher: ", err) // TODO
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
			trackedWatcher.Close()
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
			trackedWatcher.Close()
			f.fileWatchers[i] = watcher
			overwritten = true
		}
	}

	if !overwritten {
		f.fileWatchers = append(f.fileWatchers, watcher)
	}
	f.fileMux.Unlock()
}
