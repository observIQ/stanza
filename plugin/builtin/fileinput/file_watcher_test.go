package fileinput

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
	"go.uber.org/goleak"
	"go.uber.org/zap/zaptest"
)

func newOutputNotifier() (func(*entry.Entry) error, chan *entry.Entry) {
	c := make(chan *entry.Entry, 10)
	f := func(e *entry.Entry) error {
		c <- e
		return nil
	}

	return f, c
}

func newTestOffsetStore() (*OffsetStore, func()) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	remove := func() {
		os.RemoveAll(tempDir)
	}

	db, err := bbolt.Open(filepath.Join(tempDir, "bplogagent.db"), 0666, nil)
	if err != nil {
		panic(err)
	}

	store := &OffsetStore{
		db:     db,
		bucket: "test",
	}

	return store, remove
}

func TestFileWatcherReadsLog(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer func() {
		os.Remove(temp.Name())
	}()
	defer temp.Close()

	// Create the test OffsetStore
	store, remove := newTestOffsetStore()
	defer remove()

	// Create the watcher
	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher := &FileWatcher{
		path: temp.Name(),
		pathField: func() *entry.FieldSelector {
			var fs entry.FieldSelector = entry.FieldSelector([]string{"path"})
			return &fs
		}(),
		offset:           0,
		pollInterval:     time.Minute,
		splitFunc:        bufio.ScanLines,
		output:           outputFunc,
		fingerprintBytes: 100,
		offsetStore:      store,
		SugaredLogger:    logger,
	}

	// Start the watcher
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		err := watcher.Watch(ctx)
		assert.NoError(t, err)
		close(done)
	}()

	// Write a log
	_, err = temp.WriteString("test log\n")
	assert.NoError(t, err)

	// Expect that log to come back parsed correctly
	select {
	case entry := <-entryChan:
		assert.NotNil(t, entry.Record.(map[string]interface{})["message"])
		assert.NotNil(t, entry.Record.(map[string]interface{})["path"])
		assert.Equal(t, entry.Record.(map[string]interface{})["message"], "test log")
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for entry to be read")
	}

	// Cancel the context
	cancel()

	// Expect the watcher to exit
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}

func TestFileWatcher_ExitOnFileDelete(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer temp.Close()

	// Create the test offset store
	store, remove := newTestOffsetStore()
	defer remove()

	// Create the watcher
	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher := &FileWatcher{
		path:             temp.Name(),
		offset:           0,
		pollInterval:     20 * time.Millisecond,
		splitFunc:        bufio.ScanLines,
		output:           outputFunc,
		fingerprintBytes: 100,
		offsetStore:      store,
		SugaredLogger:    logger,
	}

	// Start the file watcher
	done := make(chan struct{})
	go func() {
		err := watcher.Watch(context.Background())
		assert.NoError(t, err)
		close(done)
	}()

	// Send a log to ensure the watcher is ready
	_, err = temp.WriteString("test log\n")
	assert.NoError(t, err)

	// Expect that the entry is read
	select {
	case <-entryChan:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for entry to be read")
	}

	// Remove the file
	err = os.Remove(temp.Name())
	assert.NoError(t, err)

	// Expect that the watcher exits
	select {
	case <-done:
	case <-time.After(time.Second):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}

func TestFileWatcher_ErrWatchOnFileNotExist(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	temp.Close()

	// Create the test offset store
	store, remove := newTestOffsetStore()
	assert.NoError(t, err)
	defer remove()

	// Create the watcher
	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, _ := newOutputNotifier()
	watcher := &FileWatcher{
		path:             temp.Name(),
		offset:           0,
		pollInterval:     time.Minute,
		splitFunc:        bufio.ScanLines,
		output:           outputFunc,
		fingerprintBytes: 100,
		offsetStore:      store,
		SugaredLogger:    logger,
	}

	// Remove the file
	err = os.Remove(temp.Name())
	assert.NoError(t, err)

	// Start the watcher, expect it to error
	done := make(chan struct{})
	go func() {
		err := watcher.Watch(context.Background())
		assert.Error(t, err)
		close(done)
	}()

	// Expect the watcher to finish
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}

func TestFileWatcher_PollingFallback(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer temp.Close()

	// Create the test offset store
	store, remove := newTestOffsetStore()
	assert.NoError(t, err)
	defer remove()

	// Create the watcher
	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher := &FileWatcher{
		path:             temp.Name(),
		offset:           0,
		pollInterval:     10 * time.Millisecond,
		splitFunc:        bufio.ScanLines,
		output:           outputFunc,
		fingerprintBytes: 100,
		offsetStore:      store,
		SugaredLogger:    logger,
	}

	// Override the underlying watcher with a do-nothing version
	watcher.watcher = &fsnotify.Watcher{}

	// Start the watcher
	done := make(chan struct{})
	go func() {
		err := watcher.Watch(context.Background())
		assert.NoError(t, err)
		close(done)
	}()

	// Write a log
	_, err = temp.WriteString("test log")
	assert.NoError(t, err)

	// Expect the log to be picked up by polling
	select {
	case <-entryChan:
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for entry to be read")
	}

	// Remove the file
	err = os.Remove(temp.Name())
	assert.NoError(t, err)

	// Expect the file removal to be picked up by polling
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}
