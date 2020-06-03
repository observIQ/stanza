package fileinput

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
	"go.uber.org/zap/zaptest"
)

func TestFileSourceImplements(t *testing.T) {
	require.Implements(t, (*plugin.Plugin)(nil), new(FileInput))
}

func newTestFileSource(t *testing.T) (source *FileInput, mockOutput *testutil.Plugin, cleanup func()) {
	mockOutput = &testutil.Plugin{}

	logger := zaptest.NewLogger(t).Sugar()

	db, cleanup := newTempDB()

	source = &FileInput{
		InputPlugin: helper.InputPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:      "testfile",
				PluginType:    "file_input",
				SugaredLogger: logger,
			},
			Output: mockOutput,
		},
		SplitFunc:        bufio.ScanLines,
		PollInterval:     50 * time.Millisecond,
		persist:          helper.NewScopedBBoltPersister(db, "testfile"),
		runningFiles:     make(map[string]struct{}),
		knownFiles:       make(map[string]*knownFileInfo),
		fileUpdateChan:   make(chan fileUpdateMessage),
		fingerprintBytes: 500,
	}

	return
}

func TestFileSource_Build(t *testing.T) {
	t.Parallel()
	mockOutput := &testutil.Plugin{}
	mockOutput.On("CanProcess").Return(true)
	mockOutput.On("ID").Return("mock")

	logger := zaptest.NewLogger(t).Sugar()
	db, cleanup := newTempDB()
	defer cleanup()

	pathField := entry.NewField("testpath")

	sourceConfig := &FileInputConfig{
		InputConfig: helper.InputConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "testfile",
				PluginType: "file_input",
			},
			OutputID: "mock",
		},
		Include: []string{"/var/log/testpath.*"},
		PollInterval: &plugin.Duration{
			Duration: 10 * time.Millisecond,
		},
		PathField: &pathField,
	}

	context := plugin.BuildContext{
		Logger:   logger,
		Database: db,
	}
	source, err := sourceConfig.Build(context)
	require.NoError(t, err)

	err = source.SetOutputs([]plugin.Plugin{mockOutput})
	require.NoError(t, err)

	fileInput := source.(*FileInput)
	require.Equal(t, fileInput.Output, mockOutput)
	require.Equal(t, fileInput.Include, []string{"/var/log/testpath.*"})
	require.Equal(t, fileInput.PathField, sourceConfig.PathField)
	require.Equal(t, fileInput.PollInterval, 10*time.Millisecond)
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

func newTempDB() (*bbolt.DB, func()) {
	dir, cleanup := newTempDir()
	db, err := bbolt.Open(filepath.Join(dir, "temp.db"), 0666, nil)
	if err != nil {
		panic(err)
	}

	return db, cleanup
}

func TestFileSource_CleanStop(t *testing.T) {
	t.Parallel()
	t.Skip(`Skipping due to goroutine leak in opencensus.
See this issue for details: https://github.com/census-instrumentation/opencensus-go/issues/1191#issuecomment-610440163`)
	// defer goleak.VerifyNone(t)

	source, mockOutput, cleanupSource := newTestFileSource(t)
	defer cleanupSource()

	_ = mockOutput

	tempDir, cleanupDir := newTempDir()
	defer cleanupDir()

	tempFile, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	source.Include = []string{tempFile.Name()}

	err = source.Start()
	require.NoError(t, err)

	source.Stop()

}

func expectedLogsTest(t *testing.T, expected []string, generator func(source *FileInput, tempdir string)) {
	tempDir, cleanupDir := newTempDir()
	defer cleanupDir()

	source, mockOutput, cleanupSource := newTestFileSource(t)
	defer cleanupSource()

	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	receivedMessages := make([]string, 0, 1000)
	logReceived := make(chan string, 1000)
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		logReceived <- args.Get(1).(*entry.Entry).Record.(string)
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
		case message := <-logReceived:
			receivedMessages = append(receivedMessages, message)
			if len(receivedMessages) == len(expected) {
				break LOOP
			}
			continue
		case <-timeout:
			require.FailNowf(t, "Timed out waiting for file source to read a log.", "Received: %#v\nExpected: %#v", receivedMessages, expected)
		}
	}

	select {
	case <-logReceived:
		t.Logf("Received: %#v\n", logReceived)
		require.FailNow(t, "Received an unexpected log")
	case <-time.After(20 * time.Millisecond):
	}

	source.Stop()
	wg.Wait()

	require.ElementsMatch(t, expected, receivedMessages)
}

func TestFileSource_SimpleWrite(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = temp.WriteString("testlog\n")
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_MultiFileSimple(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		temp2, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		require.NoError(t, err)

		_, err = temp2.WriteString("testlog2\n")
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog1",
		"testlog2",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_MoveFile(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		require.NoError(t, err)

		err = temp1.Close()
		require.NoError(t, err)

		err = os.Rename(temp1.Name(), fmt.Sprintf("%s.2", temp1.Name()))
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)
	}

	expectedMessages := []string{
		"testlog1",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_TruncateThenWrite(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		require.NoError(t, err)

		_, err = temp1.WriteString("testlog2\n")
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(200 * time.Millisecond)

		err = temp1.Truncate(0)
		require.NoError(t, err)
		temp1.Seek(0, 0)

		_, err = temp1.WriteString("testlog3\n")
		require.NoError(t, err)

	}

	expectedMessages := []string{
		"testlog1",
		"testlog2",
		"testlog3",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_CopyTruncateWriteBoth(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = temp1.WriteString("testlog1\n")
		require.NoError(t, err)
		_, err = temp1.WriteString("testlog2\n")
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(200 * time.Millisecond)

		temp2, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		_, err = io.Copy(temp1, temp2)
		require.NoError(t, err)

		// Truncate original file
		err = temp1.Truncate(0)
		temp1.Seek(0, 0)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Write to original and new file
		_, err = temp1.WriteString("testlog3\n")
		require.NoError(t, err)
		_, err = temp2.WriteString("testlog4\n")
		require.NoError(t, err)
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

func TestFileSource_OffsetsAfterRestart(t *testing.T) {
	t.Parallel()
	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write to a file
		_, err = temp1.WriteString("testlog1\n")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		// Restart the source
		err = source.Stop()
		require.NoError(t, err)
		err = source.Start()
		require.NoError(t, err)

		// Write a new log
		_, err = temp1.WriteString("testlog2\n")
		require.NoError(t, err)
	}

	// testlog1 should only show up once
	expectedMessages := []string{
		"testlog1",
		"testlog2",
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_OffsetsAfterRestart_BigFiles(t *testing.T) {
	t.Parallel()
	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write to a file
		_, err = temp1.WriteString(log1)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		// Restart the source
		err = source.Stop()
		require.NoError(t, err)
		err = source.Start()
		require.NoError(t, err)

		_, err = temp1.WriteString(log2)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)
	}

	// testlog1 should only show up once
	expectedMessages := []string{
		log1,
		log2,
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_OffsetsAfterRestart_BigFilesWrittenWhileOff(t *testing.T) {
	t.Parallel()
	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write to a file
		_, err = temp1.WriteString(log1)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		// Restart the source
		err = source.Stop()
		require.NoError(t, err)

		_, err = temp1.WriteString(log2)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)
	}

	// testlog1 should only show up once
	expectedMessages := []string{
		log1,
		log2,
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_FileMovedWhileOff_BigFiles(t *testing.T) {
	t.Parallel()
	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write to a file
		_, err = temp1.WriteString(log1)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		// Restart the source
		err = source.Stop()
		require.NoError(t, err)

		_, err = temp1.WriteString(log2)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)
		temp1.Close()

		err = os.Rename(temp1.Name(), fmt.Sprintf("%s2", temp1.Name()))
		require.NoError(t, err)

		err = source.Start()
		require.NoError(t, err)
	}

	// testlog1 should only show up once
	expectedMessages := []string{
		log1,
		log2,
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_FileMovedWhileOff_SmallFiles(t *testing.T) {
	t.Parallel()
	log1 := stringWithLength(10)
	log2 := stringWithLength(10)

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write to a file
		_, err = temp1.WriteString(log1)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Wait for the logs to be read and the offset to be set
		time.Sleep(50 * time.Millisecond)

		// Restart the source
		err = source.Stop()
		require.NoError(t, err)

		_, err = temp1.WriteString(log2)
		require.NoError(t, err)
		_, err = temp1.WriteString("\n")
		require.NoError(t, err)
		temp1.Close()

		err = os.Remove(temp1.Name())
		require.NoError(t, err)

		temp2, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Write the same log plus one
		temp2.WriteString(log1)
		temp2.WriteString("\n")
		temp2.WriteString(log2)
		temp2.WriteString("\n")

		err = source.Start()
		require.NoError(t, err)
	}

	// testlog1 should only show up once
	expectedMessages := []string{
		log1,
		log2,
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func TestFileSource_ManyLogsDelivered(t *testing.T) {
	t.Parallel()
	count := 1000
	expectedMessages := make([]string, 0, count)
	for i := 0; i < count; i++ {
		expectedMessages = append(expectedMessages, stringWithLength(100))
	}

	generate := func(source *FileInput, tempDir string) {
		temp1, err := ioutil.TempFile(tempDir, "")
		require.NoError(t, err)

		// Start the source
		err = source.Start()
		require.NoError(t, err)

		// Write lots of logs
		for _, message := range expectedMessages {
			temp1.WriteString(message)
			temp1.WriteString("\n")
		}
	}

	expectedLogsTest(t, expectedMessages, generate)
}

func stringWithLength(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
