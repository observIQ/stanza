package plugins

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher is a wrapper around `fsnotify` that periodically polls
// to mitigate issues with filesystems that don't support notify events
type FileWatcher struct {
	inode        uint64
	dev          uint64
	path         string
	file         *os.File
	offset       int64
	pollInterval time.Duration
	fileSource   *FileSource
	cancel       context.CancelFunc
}

func NewFileWatcher(path string, fileSource *FileSource, startFromBeginning bool) (*FileWatcher, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// TODO make this work for windows
	var inode uint64
	var dev uint64
	switch sys := fileInfo.Sys().(type) {
	case *syscall.Stat_t:
		inode = sys.Ino
		dev = uint64(sys.Dev)
	default:
		return nil, fmt.Errorf("cannot use fileinfo of type %T", fileInfo.Sys())
	}

	offset := func() int64 {
		if startFromBeginning {
			return 0
		}
		return fileSize
	}()

	return &FileWatcher{
		inode:        inode,
		dev:          dev,
		path:         path,
		pollInterval: 3 * time.Second,
		offset:       offset,
		fileSource:   fileSource,
	}, nil
}

func (w *FileWatcher) Watch(startCtx context.Context) error {
	ctx, cancel := context.WithCancel(startCtx)
	w.cancel = cancel
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %s", err)
	}

	err = watcher.Add(w.path)
	if err != nil {
		return err
	}

	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer file.Close()
	w.file = file

	for {
		// TODO actually test all these cases
		// TODO actually test all these cases on every OS we support
		// TODO actually test all these cases on weird filesystems (NFS, FUSE, etc)

		// TODO reuse the timer? but be careful about draining -- see timer.Reset() docs
		timer := time.NewTimer(w.pollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()
			err := watcher.Close()
			if err != nil {
				return err
			}
		case event, ok := <-watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			// println("Filewatcher: ", event.String(), " ", event.Name)
			if event.Op&fsnotify.Remove > 0 {
				watcher.Close()
				w.fileSource.fileRemoved <- w
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Chmod) > 0 {
				w.fileSource.fileTouched <- w.path
				fileInfo, err := file.Stat()
				if err != nil {
					return err
				}
				if fileInfo.Size() < w.offset {
					w.offset = 0
				} else if fileInfo.Size() > w.offset {
					w.offset = fileInfo.Size()
				} else {
				}
			}
			// ignore chmod and rename (rename is covered by directory create)
		case <-timer.C:
			// TODO check if the file still exists and if it grew
		case err := <-watcher.Errors:
			timer.Stop()
			// TODO should we exit?
			return err
		}
	}
}

func (w *FileWatcher) Read() {}

func (w *FileWatcher) Close() {
	if w.cancel != nil {
		w.cancel()
	}
}

type DirectoryWatcher struct {
	path         string
	watcher      *fsnotify.Watcher
	pollInterval time.Duration
	fileSource   *FileSource
}

func NewDirectoryWatcher(path string, fileSource *FileSource) (*DirectoryWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %s", err)
	}

	err = w.Add(path)
	if err != nil {
		return nil, err
	}

	return &DirectoryWatcher{
		path:         path,
		watcher:      w,
		pollInterval: 3 * time.Second,
		fileSource:   fileSource,
	}, err
}

func (w *DirectoryWatcher) Watch(ctx context.Context) error {
	for {
		timer := time.NewTimer(w.pollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()
			err := w.watcher.Close()
			if err != nil {
				return err
			}
		case event, ok := <-w.watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			// println("Dirwatcher: ", event.String(), event.Name)
			if event.Op&fsnotify.Create > 0 {
				w.fileSource.fileCreated <- event.Name
				continue
			}
			// TODO catch directory removal
			// ignore all other events since they're handled by the fileWatcher
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
