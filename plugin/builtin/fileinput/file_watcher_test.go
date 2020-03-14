package fileinput

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/stretchr/testify/assert"
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

func TestFileWatcherReadsLog(t *testing.T) {
	defer goleak.VerifyNone(t)

	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer temp.Close()

	logger := zaptest.NewLogger(t).Sugar()
	outputFunc, entryChan := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		err := watcher.Watch(ctx)
		assert.NoError(t, err)
		close(done)
	}()

	_, err = temp.WriteString("test log\n")
	assert.NoError(t, err)

	select {
	case entry := <-entryChan:
		assert.NotNil(t, entry.Record["message"])
		assert.NotNil(t, entry.Record["path"])
		assert.Equal(t, entry.Record["message"], "test log")
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for entry to be read")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}

func TestFileWatcher_ExitOnFileDelete(t *testing.T) {
	defer goleak.VerifyNone(t)

	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	defer temp.Close()
	logger := zaptest.NewLogger(t).Sugar()

	outputFunc, entryChan := newOutputNotifier()

	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		err := watcher.Watch(context.Background())
		assert.NoError(t, err)
		close(done)
	}()

	// Send a log to ensure the watcher is ready
	_, err = temp.WriteString("test log\n")
	assert.NoError(t, err)

	select {
	case <-entryChan:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for entry to be read")
	}

	err = os.Remove(temp.Name())
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}

func TestFileWatcher_ErrNewOnFileNotExist(t *testing.T) {
	defer goleak.VerifyNone(t)

	logger := zaptest.NewLogger(t)
	outputFunc, _ := newOutputNotifier()
	watcher, err := NewFileWatcher("filedoesnotexist", outputFunc, true, bufio.ScanLines, time.Minute, logger.Sugar())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}

func TestFileWatcher_ErrWatchOnFileNotExist(t *testing.T) {
	defer goleak.VerifyNone(t)

	temp, err := ioutil.TempFile("", t.Name())
	assert.NoError(t, err)
	temp.Close()

	logger := zaptest.NewLogger(t)
	outputFunc, _ := newOutputNotifier()
	watcher, err := NewFileWatcher(temp.Name(), outputFunc, true, bufio.ScanLines, time.Minute, logger.Sugar())
	assert.NoError(t, err)

	err = os.Remove(temp.Name())
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		err := watcher.Watch(context.Background())
		assert.Error(t, err)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for watcher to finish")
	}
}
