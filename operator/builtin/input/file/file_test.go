package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestFileSource(t *testing.T) (*InputOperator, chan *entry.Entry) {
	mockOutput := testutil.NewMockOperator("output")
	receivedEntries := make(chan *entry.Entry, 1000)
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedEntries <- args.Get(1).(*entry.Entry)
	})

	cfg := NewInputConfig("testfile")
	cfg.PollInterval = operator.Duration{Duration: 50 * time.Millisecond}
	cfg.StartAt = "beginning"
	cfg.Include = []string{"should-be-overwritten"}

	pg, err := cfg.Build(testutil.NewBuildContext(t))
	if err != nil {
		t.Fatalf("Error building operator: %s", err)
	}
	source := pg.(*InputOperator)
	source.OutputOperators = []operator.Operator{mockOutput}

	return source, receivedEntries
}

func TestFileSource_Build(t *testing.T) {
	t.Parallel()
	mockOutput := testutil.NewMockOperator("mock")

	basicConfig := func() *InputConfig {
		cfg := NewInputConfig("testfile")
		cfg.OutputIDs = []string{"mock"}
		cfg.Include = []string{"/var/log/testpath.*"}
		cfg.Exclude = []string{"/var/log/testpath.ex*"}
		cfg.PollInterval = operator.Duration{Duration: 10 * time.Millisecond}
		cfg.FilePathField = entry.NewRecordField("testpath")
		return cfg
	}

	cases := []struct {
		name             string
		modifyBaseConfig func(*InputConfig)
		errorRequirement require.ErrorAssertionFunc
		validate         func(*testing.T, *InputOperator)
	}{
		{
			"Basic",
			func(f *InputConfig) { return },
			require.NoError,
			func(t *testing.T, f *InputOperator) {
				require.Equal(t, f.OutputOperators[0], mockOutput)
				require.Equal(t, f.Include, []string{"/var/log/testpath.*"})
				require.Equal(t, f.FilePathField, entry.NewRecordField("testpath"))
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
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"MultilineConfiguredEndPattern",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineEndPattern: "END.*",
				}
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
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

			err = plg.SetOutputs([]operator.Operator{mockOutput})
			require.NoError(t, err)

			fileInput := plg.(*InputOperator)
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

func TestFileSource_AddFields(t *testing.T) {
	t.Parallel()
	source, logReceived := newTestFileSource(t)
	tempDir := testutil.NewTempDir(t)
	source.Include = []string{fmt.Sprintf("%s/*", tempDir)}
	source.FilePathField = entry.NewLabelField("path")
	source.FileNameField = entry.NewLabelField("file_name")

	// Create a file, then start
	temp, err := ioutil.TempFile(tempDir, "")
	require.NoError(t, err)
	defer temp.Close()

	_, err = temp.WriteString("testlog\n")
	require.NoError(t, err)

	err = source.Start()
	require.NoError(t, err)
	defer source.Stop()

	select {
	case e := <-logReceived:
		require.Equal(t, filepath.Base(temp.Name()), e.Labels["file_name"])
		require.Equal(t, temp.Name(), e.Labels["path"])
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for message")
	}
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

func waitForMessage(t *testing.T, c chan *entry.Entry, expected string) {
	select {
	case e := <-c:
		require.Equal(t, expected, e.Record.(string))
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for message")
	}
}

func waitForMessages(t *testing.T, c chan *entry.Entry, expected []string) {
	receivedMessages := make([]string, 0, 100)
LOOP:
	for {
		select {
		case e := <-c:
			receivedMessages = append(receivedMessages, e.Record.(string))
			if len(receivedMessages) == len(expected) {
				break LOOP
			}
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for expected messages")
		}
	}

	require.ElementsMatch(t, expected, receivedMessages)
}

func expectNoMessages(t *testing.T, c chan *entry.Entry) {
	select {
	case e := <-c:
		require.FailNow(t, "Received unexpected message", "Message: %s", e.Record.(string))
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEncodings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		contents []byte
		encoding string
		expected [][]byte
	}{
		{
			"Nop",
			[]byte{0xc5, '\n'},
			"",
			[][]byte{{0xc5}},
		},
		{
			"InvalidUTFReplacement",
			[]byte{0xc5, '\n'},
			"utf8",
			[][]byte{{0xef, 0xbf, 0xbd}},
		},
		{
			"ValidUTF8",
			[]byte("foo\n"),
			"utf8",
			[][]byte{[]byte("foo")},
		},
		{
			"ChineseCharacter",
			[]byte{230, 138, 152, '\n'}, // 折\n
			"utf8",
			[][]byte{{230, 138, 152}},
		},
		{
			"SmileyFaceUTF16",
			[]byte{216, 61, 222, 0, 0, 10}, // 😀\n
			"utf-16be",
			[][]byte{{240, 159, 152, 128}},
		},
		{
			"SmileyFaceNewlineUTF16",
			[]byte{216, 61, 222, 0, 0, 10, 0, 102, 0, 111, 0, 111}, // 😀\nfoo
			"utf-16be",
			[][]byte{{240, 159, 152, 128}, {102, 111, 111}},
		},
		{
			"SmileyFaceNewlineUTF16LE",
			[]byte{61, 216, 0, 222, 10, 0, 102, 0, 111, 0, 111, 0}, // 😀\nfoo
			"utf-16le",
			[][]byte{{240, 159, 152, 128}, {102, 111, 111}},
		},
		{
			"ChineseCharacterBig5",
			[]byte{167, 233, 10}, // 折\n
			"big5",
			[][]byte{{230, 138, 152}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := testutil.NewTempDir(t)
			path := filepath.Join(tempDir, "in.log")
			err := ioutil.WriteFile(path, tc.contents, 0777)
			require.NoError(t, err)

			source, receivedEntries := newTestFileSource(t)
			source.Include = []string{path}
			source.encoding, err = lookupEncoding(tc.encoding)
			require.NoError(t, err)
			source.SplitFunc, err = NewNewlineSplitFunc(source.encoding)
			require.NoError(t, err)
			require.NotNil(t, source.encoding)

			err = source.Start()
			require.NoError(t, err)

			for _, expected := range tc.expected {
				select {
				case entry := <-receivedEntries:
					require.Equal(t, expected, []byte(entry.Record.(string)))
				case <-time.After(time.Second):
					require.FailNow(t, "Timed out waiting for entry to be read")
				}
			}
		})
	}
}
