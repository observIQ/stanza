package fileinput

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
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

func TestFileWatcherReadsLog(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer temp.Close()

	// Create the watcher
	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger)
	assert.NoError(t, err)

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
		assert.NotNil(t, entry.Record["message"])
		assert.NotNil(t, entry.Record["path"])
		assert.Equal(t, entry.Record["message"], "test log")
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

	// Create the file watcher
	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel)).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger)
	assert.NoError(t, err)

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

func TestFileWatcher_ErrNewOnFileNotExist(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Expect creating a file watcher to fail if the file does not exist
	logger := zaptest.NewLogger(t)
	outputFunc, _ := newOutputNotifier()
	watcher, err := NewFileWatcher("filedoesnotexist", outputFunc, true, bufio.ScanLines, time.Minute, logger.Sugar())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}

func TestFileWatcher_ErrWatchOnFileNotExist(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create the temp file
	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	temp.Close()

	// Create the file watcher
	logger := zaptest.NewLogger(t)
	outputFunc, _ := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger.Sugar())
	assert.NoError(t, err)

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

	// Create the file watcher with low poll rate
	logger := zaptest.NewLogger(t)
	outputFunc, entryChan := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, 10*time.Millisecond, logger.Sugar())
	assert.NoError(t, err)

	// Override the underlying watcher with a do-nothing version
	// TODO a nicer way to inject the watcher dependency would make things much cleaner
	watcher.watcher.Close()
	watcher.watcher = &fsnotify.Watcher{}
	watcher.pollingOnly = true

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
