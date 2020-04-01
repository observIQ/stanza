package fileinput

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
	"go.uber.org/zap/zaptest"
)

func TestFileSourceImplements(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(FileInput))
	assert.Implements(t, (*plugin.Producer)(nil), new(FileInput))
}

func newTestFileSource(t *testing.T) (source *FileInput, mockConsumer *testutil.Consumer, closeStore func()) {
	mockConsumer = &testutil.Consumer{}

	var offsetStore *OffsetStore
	offsetStore, closeStore = newTestOffsetStore()

	logger := zaptest.NewLogger(t).Sugar()

	source = &FileInput{
		InputPlugin: base.InputPlugin{
			Plugin: base.Plugin{
				PluginID:      "testfile",
				PluginType:    "file",
				SugaredLogger: logger,
			},
			Output: nil,
		},
		SplitFunc:        bufio.ScanLines,
		PollInterval:     1 * time.Minute,
		FingerprintBytes: 100,

		fileCreated: make(chan string),
		offsetStore: offsetStore,
	}

	return
}

func newTempDir() (tempDir string, cleanup func()) {
	var err error
	tempDir, err = ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	return
}

func TestFileSource_CleanStop(t *testing.T) {
	defer goleak.VerifyNone(t)

	source, mockConsumer, cleanupSource := newTestFileSource(t)
	defer cleanupSource()

	_ = mockConsumer

	tempDir, cleanupDir := newTempDir()
	defer cleanupDir()

	tempFile, err := ioutil.TempFile(tempDir, "")
	assert.NoError(t, err)

	source.Include = []string{tempFile.Name()}

	err = source.Start()
	assert.NoError(t, err)

	source.Stop()

}

func expectedLogsTest(t *testing.T, expected []string, generator func(source *FileInput, tempdir string)) {
	defer goleak.VerifyNone(t)

	source, mockConsumer, cleanupSource := newTestFileSource(t)
	defer cleanupSource()

	tempDir, cleanupDir := newTempDir()
	defer cleanupDir()

	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	receivedMessages := make([]string, 0)
	logReceived := make(chan struct{})
	mux := &sync.Mutex{}
	mockConsumer.On("Consume", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		mux.Lock()
		receivedMessages = append(receivedMessages, args.Get(0).(*entry.Entry).Record["message"].(string))
		logReceived <- struct{}{}
		mux.Unlock()
	})

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		generator(source, tempDir)
	}()

	timeout := time.After(5 * time.Second)
LOOP:
	for {
		select {
		case <-logReceived:
			if len(receivedMessages) == len(expected) {
				break LOOP
			}
			continue
		case <-timeout:
			assert.FailNowf(t, "Timed out waiting for file source to read a log.", "Received: %#v\nExpected: %#v", receivedMessages, expected)
		}
	}

	select {
	case <-logReceived:
		assert.FailNowf(t, "Received an unexpected log", "Received: %#v\nExpected: %#v", receivedMessages, expected)
	case <-time.After(50 * time.Millisecond):
	}

	source.Stop()
	wg.Wait()

	if !assert.ElementsMatch(t, expected, receivedMessages) {
		t.Logf("Received: %#v\n", receivedMessages)
	}
}

func TestFileSource_SimpleWrite(t *testing.T) {
	defer goleak.VerifyNone(t)

	generate := func(source *FileInput, tempDir string) {
		temp, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = temp.WriteString("testlog\n")
		assert.NoError(t, err)

		err = source.Start()
		assert.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_MultiFileSimple(t *testing.T) {
	defer goleak.VerifyNone(t)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		temp2, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		assert.NoError(t, err)

		_, err = temp2.WriteString("testlog2\n")
		assert.NoError(t, err)

		err = source.Start()
		assert.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog1",
		"testlog2",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_MoveFile(t *testing.T) {
	defer goleak.VerifyNone(t)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		assert.NoError(t, err)

		err = os.Rename(temp1.Name(), fmt.Sprintf("%s.2", temp1.Name()))
		assert.NoError(t, err)

		err = source.Start()
		assert.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog1",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_TruncateThenWrite(t *testing.T) {
	defer goleak.VerifyNone(t)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog2\n")
		assert.NoError(t, err)

		err = source.Start()
		assert.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		err = temp1.Truncate(0)
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog3\n")
		assert.NoError(t, err)

	}

	expectedMessages := []string{
		"testlog1",
		"testlog2",
		"testlog3",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_CopyTruncateWriteBoth(t *testing.T) {
	defer goleak.VerifyNone(t)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		assert.NoError(t, err)

		_, err = temp1.WriteString("testlog2\n")
		assert.NoError(t, err)

		err = source.Start()
		assert.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		temp2, err := ioutil.TempFile(tempDir, "")
		assert.NoError(t, err)

		_, err = io.Copy(temp1, temp2)
		assert.NoError(t, err)

		// Truncate original file
		err = temp1.Truncate(0)
		temp1.Seek(0, 0)
		assert.NoError(t, err)

		// Write to original and new file
		_, err = temp1.WriteString("testlog3\n")
		assert.NoError(t, err)
		_, err = temp2.WriteString("testlog4\n")
		assert.NoError(t, err)

	}

	// testlog1 and testlog2 should only show up once
	expectedMessages := []string{
		"testlog1",
		"testlog2",
		"testlog3",
		"testlog4",
	}

	expectedLogsTest(t, expectedMessages, generate)
}
