package fileinput

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// FileWatcher is a wrapper around `fsnotify` that periodically polls to provide
// a fallback for filesystems and platforms that don't support event notification
type FileWatcher struct {
	inode        uint64
	dev          uint64
	path         string
	file         *os.File
	offset       int64
	pollInterval time.Duration
	cancel       context.CancelFunc
	splitFunc    bufio.SplitFunc
	output       func(*entry.Entry) error

	*zap.SugaredLogger
}

func NewFileWatcher(
	path string,
	output func(*entry.Entry) error,
	startFromBeginning bool,
	splitFunc bufio.SplitFunc,
	pollInterval time.Duration,
	logger *zap.SugaredLogger,
) (*FileWatcher, error) {

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
		inode:         inode,
		dev:           dev,
		path:          path,
		pollInterval:  pollInterval,
		offset:        offset,
		splitFunc:     splitFunc,
		output:        output,
		SugaredLogger: logger.Named("file_watcher").With("path", path),
	}, nil
}

func (w *FileWatcher) Watch(startCtx context.Context) error {
	ctx, cancel := context.WithCancel(startCtx)
	w.cancel = cancel

	// Create the watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// TODO if falling back to polling, should we set the default lower?
		w.Infow("Failed to create notifying watcher. Falling back to polling only", "error", err)
		watcher = &fsnotify.Watcher{} // create an empty watcher whose channels are just nil
	} else {
		defer watcher.Close()
		err = watcher.Add(w.path)
		if err != nil {
			w.Infow("Failed to add path to watcher. Falling back to polling only", "error", err)
			watcher = &fsnotify.Watcher{} // create an empty watcher whose channels are just nil
		}
	}

	// Keep a persistent open file
	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer file.Close()
	w.file = file

	// Check it once initially for responsive startup
	err = w.checkReadFile(ctx)
	if err != nil {
		return err
	}

	for {
		// TODO actually test all these cases
		// TODO actually test all these cases on every OS we support
		// TODO actually test all these cases on weird filesystems (NFS, FUSE, etc)

		// TODO reuse the timer? but be careful about draining -- see timer.Reset() docs
		timer := time.NewTimer(w.pollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case event, ok := <-watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Remove > 0 {
				return watcher.Close()
			}
			if event.Op&(fsnotify.Write|fsnotify.Chmod) > 0 {
				err := w.checkReadFile(ctx)
				if err != nil {
					return err
				}
			}
			// ignore rename (rename is covered by directory create)
		case <-timer.C:
			err := w.checkReadFile(ctx)
			if err != nil {
				return err
			}
		case err := <-watcher.Errors:
			timer.Stop()
			return err
		}
	}
}

func (w *FileWatcher) checkReadFile(ctx context.Context) error {
	// TODO ensure that none of the errors thrown in here are recoverable
	// since returning an error triggers a return from the watch function
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	// TODO check if the file still exists

	fileInfo, err := w.file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() < w.offset {
		w.Debug("Detected file truncation. Starting from beginning")
		w.offset, err = w.file.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("seek to start: %s", err)
		}
		err := w.readToEnd(ctx)
		if err != nil {
			return err
		}
	} else if fileInfo.Size() > w.offset {
		err := w.readToEnd(ctx)
		if err != nil {
			return err
		}
	}

	// do nothing if the file hasn't changed size
	return nil
}

func (w *FileWatcher) readToEnd(ctx context.Context) error {
	// TODO seek to last offset?
	scanner := bufio.NewScanner(w.file)
	scanner.Split(w.splitFunc)
	// TODO scanner.Buffer() to set max size

	for {
		select {
		case <-ctx.Done():
			return nil // Stop reading if closed
		default:
		}

		ok := scanner.Scan()
		if !ok {
			return scanner.Err()
		}

		message := scanner.Text()
		entry := &entry.Entry{
			Timestamp: time.Now(),
			Record: map[string]interface{}{
				"message": message,
				"path":    w.path, // TODO use absolute path?
			},
		}

		err := w.output(entry)
		if err != nil {
			return fmt.Errorf("output entry: %s", err)
		}

		// TODO does this actually work how I think it does with the scanner?
		// I'm unsure if the scanner peeks ahead, or actually advances the reader
		// every time it tries to parse something. This needs to be tested
		w.offset, err = w.file.Seek(0, 1) // get current file offset
		if err != nil {
			return fmt.Errorf("get current offset: %s", err)
		}
	}
}

func (w *FileWatcher) Close() {
	if w.cancel != nil {
		w.cancel()
	}
}
