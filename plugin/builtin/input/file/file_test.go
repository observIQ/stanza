package file

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestFileSource(t *testing.T) (*InputPlugin, chan string) {
	mockOutput := testutil.NewMockPlugin("output")
	receivedMessages := make(chan string, 1000)
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedMessages <- args.Get(1).(*entry.Entry).Record.(string)
	})

	logger := zaptest.NewLogger(t).Sugar()
	db := testutil.NewTestDatabase(t)

	source := &InputPlugin{
		InputPlugin: helper.InputPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:      "testfile",
				PluginType:    "file_input",
				SugaredLogger: logger,
			},
			WriterPlugin: helper.WriterPlugin{
				OutputPlugins: []plugin.Plugin{mockOutput},
			},
			WriteTo: entry.NewRecordField(),
		},
		SplitFunc:        bufio.ScanLines,
		PollInterval:     50 * time.Millisecond,
		persist:          helper.NewScopedDBPersister(db, "testfile"),
		runningFiles:     make(map[string]struct{}),
		knownFiles:       make(map[string]*knownFileInfo),
		fileUpdateChan:   make(chan fileUpdateMessage),
		fingerprintBytes: 500,
		startAtBeginning: true,
	}

	return source, receivedMessages
}

func TestFileSource_Build(t *testing.T) {
	t.Parallel()
	mockOutput := testutil.NewMockPlugin("mock")

	pathField := entry.NewRecordField("testpath")

	basicConfig := func() *InputConfig {
		return &InputConfig{
			InputConfig: helper.InputConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "testfile",
					PluginType: "file_input",
				},
				WriterConfig: helper.WriterConfig{
					OutputIDs: []string{"mock"},
				},
				WriteTo: entry.NewRecordField(),
			},
			Include: []string{"/var/log/testpath.*"},
			Exclude: []string{"/var/log/testpath.ex*"},
			PollInterval: &plugin.Duration{
				Duration: 10 * time.Millisecond,
			},
			PathField: &pathField,
		}
	}

	cases := []struct {
		name             string
		modifyBaseConfig func(*InputConfig)
		errorRequirement require.ErrorAssertionFunc
		validate         func(*testing.T, *InputPlugin)
	}{
		{
			"Basic",
			func(f *InputConfig) { return },
			require.NoError,
			func(t *testing.T, f *InputPlugin) {
				require.Equal(t, f.OutputPlugins[0], mockOutput)
				require.Equal(t, f.Include, []string{"/var/log/testpath.*"})
				require.Equal(t, f.PathField, &pathField)
				require.Equal(t, f.PollInterval, 10*time.Millisecond)
			},
		},
		{
			"BadIncludeGlob",
			func(f *InputConfig) {
				f.Include = []string{"["}
			},
			require.Error,
			nil,
		},
		{
			"BadExcludeGlob",
			func(f *InputConfig) {
				f.Include = []string{"["}
			},
			require.Error,
			nil,
		},
		{
			"MultilineConfiguredStartAndEndPatterns",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineEndPattern:   "Exists",
					LineStartPattern: "Exists",
				}
			},
			require.Error,
			nil,
		},
		{
			"MultilineConfiguredStartPattern",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineStartPattern: "START.*",
				}
			},
			require.NoError,
			func(t *testing.T, f *InputPlugin) {},
		},
		{
			"MultilineConfiguredEndPattern",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineEndPattern: "END.*",
				}
			},
			require.NoError,
			func(t *testing.T, f *InputPlugin) {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := basicConfig()
			tc.modifyBaseConfig(cfg)

			plg, err := cfg.Build(testutil.NewBuildContext(t))
			tc.errorRequirement(t, err)
			if err != nil {
				return
			}

			err = plg.SetOutputs([]plugin.Plugin{mockOutput})
			require.NoError(t, err)

			fileInput := plg.(*InputPlugin)
			tc.validate(t, fileInput)
		})
	}
}

func TestFileSource_CleanStop(t *testing.T) {
	t.Parallel()
	t.Skip(`Skipping due to goroutine leak in opencensus.
See this issue for details: https://github.com/census-instrumentation/opencensus-go/issues/1191#issuecomment-610440163`)
	// defer goleak.VerifyNone(t)

	source, _ := newTestFileSource(t)

	tempDir := testutil.NewTempDir(t)

	tempFile, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	source.Include = []string{tempFile.Name()}

	err = source.Start()
	require.NoError(t, err)
	source.Stop()
}

func TestFileSource_ReadExistingLogs(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	// Create a file, then start
	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog")
}

func TestFileSource_ReadNewLogs(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	// Start first, then create a new file
	err := source.Start()
	require.NoError(t, err)
	defer source.Stop()

	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog")
}

func TestFileSource_ReadExistingAndNewLogs(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog1\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	_, err = temp.WriteString("testlog2\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

func TestFileSource_StartAtEnd(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}
	source.startAtBeginning = false

	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog1\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	// Wait until file has been read the first time
	time.Sleep(200 * time.Millisecond)

	_, err = temp.WriteString("testlog2\n")
	require.NoError(t, err)
	temp.Close()

	waitForMessage(t, logReceived, "testlog2")
}

func TestFileSource_StartAtEndNewFile(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}
	source.startAtBeginning = false

	err := source.Start()
	require.NoError(t, err)
	defer source.Stop()

	// Wait for the first check to complete
	time.Sleep(200 * time.Millisecond)

	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog1\ntestlog2\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

func TestFileSource_MultiFileSimple(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

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
	defer source.Stop()

	waitForMessages(t, logReceived, []string{"testlog1", "testlog2"})
}

func TestFileSource_MoveFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Moving files while open is unsupported on Windows")
	}
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	_, err = temp1.WriteString("testlog1\n")
	require.NoError(t, err)
	temp1.Close()

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	time.Sleep(200 * time.Millisecond)

	i := 0
	for {
		err = os.Rename(temp1.Name(), fmt.Sprintf("%s.2", temp1.Name()))
		if err != nil {
			if i < 3 {
				t.Error(err)
				i++
				time.Sleep(10 * time.Millisecond)
				continue
			} else {
				require.NoError(t, err)
			}
		}

		break
	}

	expectNoMessages(t, logReceived)
}

func TestFileSource_TruncateThenWrite(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	_, err = temp1.WriteString("testlog1\n")
	require.NoError(t, err)
	_, err = temp1.WriteString("testlog2\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")

	err = temp1.Truncate(0)
	require.NoError(t, err)
	temp1.Seek(0, 0)

	_, err = temp1.WriteString("testlog3\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog3")
	expectNoMessages(t, logReceived)
}

func TestFileSource_CopyTruncateWriteBoth(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp1.Close()

	_, err = temp1.WriteString("testlog1\n")
	require.NoError(t, err)
	_, err = temp1.WriteString("testlog2\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")

	time.Sleep(50 * time.Millisecond)

	temp2, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp2.Close()

	_, err = io.Copy(temp1, temp2)
	require.NoError(t, err)

	// Truncate original file
	err = temp1.Truncate(0)
	temp1.Seek(0, 0)
	require.NoError(t, err)

	// Write to original and new file
	_, err = temp1.WriteString("testlog3\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog3")

	_, err = temp2.WriteString("testlog4\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog4")
}

func TestFileSource_OffsetsAfterRestart(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	// Write to a file
	_, err = temp1.WriteString("testlog1\n")
	require.NoError(t, err)

	// Start the source
	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")

	// Restart the source
	err = source.Stop()
	require.NoError(t, err)
	err = source.Start()
	require.NoError(t, err)

	// Write a new log
	_, err = temp1.WriteString("testlog2\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, "testlog2")
}

func TestFileSource_OffsetsAfterRestart_BigFiles(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	// Write to a file
	_, err = temp1.WriteString(log1 + "\n")
	require.NoError(t, err)

	// Start the source
	err = source.Start()
	require.NoError(t, err)

	waitForMessage(t, logReceived, log1)

	// Restart the source
	err = source.Stop()
	require.NoError(t, err)
	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	_, err = temp1.WriteString(log2 + "\n")
	require.NoError(t, err)

	waitForMessage(t, logReceived, log2)
}

func TestFileSource_OffsetsAfterRestart_BigFilesWrittenWhileOff(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	// Write to a file
	_, err = temp1.WriteString(log1 + "\n")
	require.NoError(t, err)

	// Start the source
	err = source.Start()
	require.NoError(t, err)

	waitForMessage(t, logReceived, log1)

	// Restart the source
	err = source.Stop()
	require.NoError(t, err)

	_, err = temp1.WriteString(log2 + "\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, log2)
}

func TestFileSource_FileMovedWhileOff_BigFiles(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	// Write to a file
	_, err = temp1.WriteString(log1 + "\n")
	require.NoError(t, err)

	// Start the source
	err = source.Start()
	require.NoError(t, err)

	waitForMessage(t, logReceived, log1)

	// Stop the source, then rename and write a new log
	err = source.Stop()
	require.NoError(t, err)

	_, err = temp1.WriteString(log2 + "\n")
	require.NoError(t, err)
	temp1.Close()

	err = os.Rename(temp1.Name(), fmt.Sprintf("%s2", temp1.Name()))
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, log2)
}

func TestFileSource_FileMovedWhileOff_SmallFiles(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	log1 := stringWithLength(10)
	log2 := stringWithLength(10)

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	// Write to a file
	_, err = temp1.WriteString(log1 + "\n")
	require.NoError(t, err)

	// Start the source
	err = source.Start()
	require.NoError(t, err)

	waitForMessage(t, logReceived, log1)

	// Restart the source
	err = source.Stop()
	require.NoError(t, err)

	_, err = temp1.WriteString(log2 + "\n")
	require.NoError(t, err)
	temp1.Close()

	err = os.Rename(temp1.Name(), fmt.Sprintf("%s2", temp1.Name()))
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	waitForMessage(t, logReceived, log2)
}

func TestFileSource_ManyLogsDelivered(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}

	temp1, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)

	count := 1000
	expectedMessages := make([]string, 0, count)
	for i := 0; i < count; i++ {
		expectedMessages = append(expectedMessages, strconv.Itoa(i))
	}

	// Start the source
	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	// Write lots of logs
	for _, message := range expectedMessages {
		temp1.WriteString(message + "\n")
	}

	// Expect each of them to come through
	for _, message := range expectedMessages {
		waitForMessage(t, logReceived, message)
	}

	expectNoMessages(t, logReceived)
}

func stringWithLength(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func waitForMessage(t *testing.T, c chan string, expected string) {
	select {
	case m := <-c:
		require.Equal(t, expected, m)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for message")
	}
}

func waitForMessages(t *testing.T, c chan string, expected []string) {
	receivedMessages := make([]string, 0, 100)
LOOP:
	for {
		select {
		case m := <-c:
			receivedMessages = append(receivedMessages, m)
			if len(receivedMessages) == len(expected) {
				break LOOP
			}
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for expected messages")
		}
	}

	require.ElementsMatch(t, expected, receivedMessages)
}

func expectNoMessages(t *testing.T, c chan string) {
	select {
	case m := <-c:
		require.FailNow(t, "Received unexpected message", "Message: %s", m)
	case <-time.After(200 * time.Millisecond):
	}
}
