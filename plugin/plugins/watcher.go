package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher is a wrapper around `fsnotify` that periodically polls
// to mitigate issues with filesystems that don't support notify events
type FileWatcher struct {
	path         string
	file         *os.File
	watcher      *fsnotify.Watcher
	offset       int
	pollInterval time.Duration
}

func NewFileWatcher(path string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %s", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("determining absolute path: %s", err)
	}

	return &FileWatcher{
		path:         absPath,
		watcher:      w,
		pollInterval: 3 * time.Second,
	}, err
}

func (w *FileWatcher) Watch(ctx context.Context) error {
	for {
		// TODO actually test all these cases
		// TODO actually test all these cases on every OS we support
		// TODO actually test all these cases on weird filesystems (NFS, FUSE, etc)

		// TODO reuse the timer? but be careful about draining -- see timer.Reset() docs
		timer := time.NewTimer(w.pollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()
			w.watcher.Close()
		case event, ok := <-w.watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			fmt.Printf("File event: %s\n", event)
		case <-timer.C:
			// Try to read file anyways
			w.Read()
		case err := <-w.watcher.Errors:
			timer.Stop()
			// TODO should we exit?
			return err
		}

	}
}

func (w *FileWatcher) Read() {}

type DirectoryWatcher struct {
	path         string
	watcher      *fsnotify.Watcher
	pollInterval time.Duration
}

func NewDirectoryWatcher(path string, newFileCallback func(path string)) (*DirectoryWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %s", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("determining absolute path: %s", err)
	}

	return &DirectoryWatcher{
		path:         absPath,
		watcher:      w,
		pollInterval: 3 * time.Second,
	}, err
}

func (w *DirectoryWatcher) Watch(ctx context.Context) error {
	for {
		timer := time.NewTimer(w.pollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()
			w.watcher.Close()
		case event, ok := <-w.watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			fmt.Printf("Directory event: %s\n", event)
		case <-timer.C:
		case err := <-w.watcher.Errors:
			timer.Stop()
			// TODO should we exit?
			return err
		}

	}
}

func (w *DirectoryWatcher) Close() {
	w.watcher.Close()
}
