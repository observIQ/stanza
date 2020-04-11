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
	path             string
	pathField        *entry.FieldSelector
	offset           int64
	pollInterval     time.Duration
	splitFunc        bufio.SplitFunc
	output           func(*entry.Entry) error
	fingerprintBytes int64
	offsetStore      *OffsetStore
	*zap.SugaredLogger

	cancel            context.CancelFunc
	stableFingerprint []byte
	watcher           *fsnotify.Watcher
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
	// Return an error the first time if reading fails
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
					return nil
				}
			}
			// ignore rename (rename is covered by directory create)
		case <-timer.C:
			err := w.checkReadFile(ctx)
			if err != nil {
				return nil
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
		w.Exit()
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
		w.stableFingerprint = nil
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

	// Create an intermediary scan func that keeps track of how far we've advanced
	pos := w.offset
	scanFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		pos += int64(advance)
		return
	}
	scanner.Split(scanFunc)
	// TODO scanner.Buffer() to set max size

	defer func() {
		// TODO this will be very slow while the fingerprint hasn't stabilized
		fingerprint, err := w.Fingerprint()
		if err != nil {
			w.Warnf("failed to get fingerprint", "error", err)
		}

		// TODO only doing this at the end of reads means that we can't guarantee
		// no duplicate logs
		err = w.offsetStore.SetOffset(fingerprint, w.offset)
		if err != nil {
			w.Warnf("failed to set offset", "error", err)
		}

		w.Debugw("Stored offset", "path", w.path, "offset", w.offset, "fingerprint", fingerprint)
	}()

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
			},
		}

		if w.pathField != nil {
			entry.Set(*w.pathField, w.path)
		}

		err := w.output(entry)
		if err != nil {
			w.Warnw("Failed to process entry", "error", err, "entry", entry)
		}

		w.offset = pos
	}
}

func (w *FileWatcher) Fingerprint() ([]byte, error) {
	if w.stableFingerprint == nil {
		file, err := os.Open(w.path) // TODO handle error
		if err != nil {
			return nil, err
		}
		defer file.Close()
		fp, stable := fingerprint(w.fingerprintBytes, file)
		if stable {
			w.stableFingerprint = fp
		}
		return fp, nil
	} else {
		return w.stableFingerprint, nil
	}
}

func (w *FileWatcher) Exit() {
	if w.cancel != nil {
		w.cancel()
	}
}
