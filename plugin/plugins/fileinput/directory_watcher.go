package fileinput

import (
	"context"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
)

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
