package fileinput

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// FileWatcher is a wrapper around `fsnotify` that periodically polls to provide
// a fallback for filesystems and platforms that don't support event notification
type FileWatcher struct {
	path   string
	offset int64

	pollInterval time.Duration

	cancel context.CancelFunc

	splitFunc   bufio.SplitFunc
	output      func(*entry.Entry) error
	watcher     *fsnotify.Watcher
	fingerprint *string

	*zap.SugaredLogger
}

func NewFileWatcher(
	path string,
	output func(*entry.Entry) error,
	offset int64,
	splitFunc bufio.SplitFunc,
	pollInterval time.Duration,
	logger *zap.SugaredLogger,
) *FileWatcher {

	return &FileWatcher{
		path: path,

		pollInterval: pollInterval,

		offset:        offset,
		splitFunc:     splitFunc,
		output:        output,
		SugaredLogger: logger.With("path", path),
	}
}

func (w *FileWatcher) Watch(startCtx context.Context) error {
	// Create the watcher if it's not manually set (probably only in tests)
	if w.watcher == nil {
		var err error
		w.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			// TODO if falling back to polling, should we set the default lower?
			// TODO maybe even make some sort of smart interval tuning?
			w.watcher = &fsnotify.Watcher{} // create an empty watcher whose channels are just nil
		} else {
			defer w.watcher.Close()
			err = w.watcher.Add(w.path)
			if err != nil {
				w.watcher = &fsnotify.Watcher{} // create an empty watcher whose channels are just nil
			}
		}
	}

	ctx, cancel := context.WithCancel(startCtx)
	w.cancel = cancel

	// Check it once initially for responsive startup
	err := w.checkReadFile(ctx)
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
		case event, ok := <-w.watcher.Events:
			timer.Stop()
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) > 0 {
				return nil
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
		case err := <-w.watcher.Errors:
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

	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() < w.offset {
		w.Info("Detected file truncation. Starting from beginning")
		w.offset = 0
		w.fingerprint = nil
		err := w.readToEnd(ctx, file)
		if err != nil {
			return err
		}
	} else if fileInfo.Size() > w.offset {
		err := w.readToEnd(ctx, file)
		if err != nil {
			return err
		}
	}

	// do nothing if the file hasn't changed size
	return nil
}

func (w *FileWatcher) readToEnd(ctx context.Context, file *os.File) error {
	var err error
	w.offset, err = file.Seek(w.offset, 0) // set the file to the last offset
	if err != nil {
		return fmt.Errorf("get current offset: %s", err)
	}

	scanner := bufio.NewScanner(file)
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
		w.offset, err = file.Seek(0, 1) // get current file offset
		if err != nil {
			return fmt.Errorf("get current offset: %s", err)
		}
	}
}

func (w *FileWatcher) Offset() int64 {
	return w.offset
}

func (w *FileWatcher) Fingerprint(numBytes int64) string {
	if w.fingerprint == nil {
		file, err := os.Open(w.path) // TODO handle error
		if err != nil {
			return err.Error()
		}
		defer file.Close()
		fp := fingerprint(numBytes, file)
		w.fingerprint = &fp
		return fp
	} else {
		return *w.fingerprint
	}
}

func (w *FileWatcher) Close() {
	if w.cancel != nil {
		w.cancel()
	}
}
