package file

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func newDefaultConfig(tempDir string) *InputConfig {
	cfg := NewInputConfig("testfile")
	cfg.PollInterval = helper.Duration{Duration: 50 * time.Millisecond}
	cfg.StartAt = "beginning"
	cfg.Include = []string{fmt.Sprintf("%s/*", tempDir)}
	cfg.OutputIDs = []string{"fake"}
	return cfg
}

func newTestFileSource(t *testing.T, cfgMod func(*InputConfig)) (*InputOperator, chan *entry.Entry, string) {
	fakeOutput := testutil.NewFakeOutput(t)
	tempDir := testutil.NewTempDir(t)

	cfg := newDefaultConfig(tempDir)
	if cfgMod != nil {
		cfgMod(cfg)
	}
	pg, err := cfg.Build(testutil.NewBuildContext(t))
	if err != nil {
		t.Fatalf("Error building operator: %s", err)
	}
	err = pg.SetOutputs([]operator.Operator{fakeOutput})
	require.NoError(t, err)

	return pg.(*InputOperator), fakeOutput.Received, tempDir
}

func openFile(t testing.TB, path string) *os.File {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0777)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	return file
}

func openTemp(t testing.TB, tempDir string) *os.File {
	return openTempWithPattern(t, tempDir, "")
}

func reopenTemp(t testing.TB, name string) *os.File {
	return openTempWithPattern(t, filepath.Dir(name), filepath.Base(name))
}

func openTempWithPattern(t testing.TB, tempDir, pattern string) *os.File {
	file, err := ioutil.TempFile(tempDir, pattern)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	return file
}

func writeString(t testing.TB, file *os.File, s string) {
	_, err := file.WriteString(s)
	require.NoError(t, err)
}

func TestFileSource_Build(t *testing.T) {
	t.Parallel()
	fakeOutput := testutil.NewMockOperator("fake")

	basicConfig := func() *InputConfig {
		cfg := NewInputConfig("testfile")
		cfg.OutputIDs = []string{"fake"}
		cfg.Include = []string{"/var/log/testpath.*"}
		cfg.Exclude = []string{"/var/log/testpath.ex*"}
		cfg.PollInterval = helper.Duration{Duration: 10 * time.Millisecond}
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
				require.Equal(t, f.OutputOperators[0], fakeOutput)
				require.Equal(t, f.Include, []string{"/var/log/testpath.*"})
				require.Equal(t, f.FilePathField, entry.NewNilField())
				require.Equal(t, f.FileNameField, entry.NewLabelField("file_name"))
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
		{
			"InvalidEncoding",
			func(f *InputConfig) {
				f.Encoding = "UTF-3233"
			},
			require.Error,
			nil,
		},
		{
			"LineStartAndEnd",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineStartPattern: ".*",
					LineEndPattern:   ".*",
				}
			},
			require.Error,
			nil,
		},
		{
			"NoLineStartOrEnd",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{}
			},
			require.Error,
			nil,
		},
		{
			"InvalidLineStartRegex",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineStartPattern: "(",
				}
			},
			require.Error,
			nil,
		},
		{
			"InvalidLineEndRegex",
			func(f *InputConfig) {
				f.Multiline = &MultilineConfig{
					LineEndPattern: "(",
				}
			},
			require.Error,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			cfg := basicConfig()
			tc.modifyBaseConfig(cfg)

			plg, err := cfg.Build(testutil.NewBuildContext(t))
			tc.errorRequirement(t, err)
			if err != nil {
				return
			}

			err = plg.SetOutputs([]operator.Operator{fakeOutput})
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

	source, _, tempDir := newTestFileSource(t, nil)
	_ = openTemp(t, tempDir)
	err := source.Start()
	require.NoError(t, err)
	source.Stop()
}

// AddFields tests that the `file_name` and `file_path` fields are included
// when IncludeFileName and IncludeFilePath are set to true
func TestFileSource_AddFileFields(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, func(cfg *InputConfig) {
		cfg.IncludeFileName = true
		cfg.IncludeFilePath = true
	})

	// Create a file, then start
	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog\n")

	require.NoError(t, source.Start())
	defer source.Stop()

	e := waitForOne(t, logReceived)
	require.Equal(t, filepath.Base(temp.Name()), e.Labels["file_name"])
	require.Equal(t, temp.Name(), e.Labels["file_path"])
}

// ReadExistingLogs tests that, when starting from beginning, we
// read all the lines that are already there
func TestFileSource_ReadExistingLogs(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	// Create a file, then start
	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\ntestlog2\n")

	require.NoError(t, source.Start())
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

// ReadNewLogs tests that, after starting, if a new file is created
// all the entries in that file are read from the beginning
func TestFileSource_ReadNewLogs(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	// Poll once so we know this isn't a new file
	source.poll(context.Background())
	defer source.Stop()

	// Create a new file
	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog\n")

	// Poll a second time after the file has been created
	source.poll(context.Background())

	// Expect the message to come through
	waitForMessage(t, logReceived, "testlog")
}

// ReadExistingAndNewLogs tests that, on startup, if start_at
// is set to `beginning`, we read the logs that are there, and
// we read any additional logs that are written after startup
func TestFileSource_ReadExistingAndNewLogs(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	// Start with a file with an entry in it, and expect that entry
	// to come through when we poll for the first time
	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\n")
	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog1")

	// Write a second entry, and expect that entry to come through
	// as well
	writeString(t, temp, "testlog2\n")
	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog2")
}

// StartAtEnd tests that when `start_at` is configured to `end`,
// we don't read any entries that were in the file before startup
func TestFileSource_StartAtEnd(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, func(cfg *InputConfig) {
		cfg.StartAt = "end"
	})

	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\n")

	// Expect no entries on the first poll
	source.poll(context.Background())
	expectNoMessages(t, logReceived)

	// Expect any new entries after the first poll
	writeString(t, temp, "testlog2\n")
	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog2")
}

// StartAtEndNewFile tests that when `start_at` is configured to `end`,
// a file created after the source has been started is read from the
// beginning
func TestFileSource_StartAtEndNewFile(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)
	source.startAtBeginning = false

	source.poll(context.Background())

	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\ntestlog2\n")

	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

// NoNewline tests that an entry will still be sent eventually
// even if the file doesn't end in a newline
func TestFileSource_NoNewline(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\ntestlog2")

	require.NoError(t, source.Start())
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

// SkipEmpty tests that the any empty lines are skipped
func TestFileSource_SkipEmpty(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1\n\ntestlog2\n")

	require.NoError(t, source.Start())
	defer source.Stop()

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
}

// SplitWrite tests a line written in two writes
// close together still is read as a single entry
func TestFileSource_SplitWrite(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp := openTemp(t, tempDir)
	writeString(t, temp, "testlog1")

	source.poll(context.Background())

	writeString(t, temp, "testlog2\n")

	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog1testlog2")
}

func TestFileSource_DecodeBufferIsResized(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	require.NoError(t, source.Start())
	defer source.Stop()

	temp := openTemp(t, tempDir)
	expected := stringWithLength(1<<12 + 1)
	writeString(t, temp, expected+"\n")

	waitForMessage(t, logReceived, expected)
}

func TestFileSource_MultiFileSimple(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp1 := openTemp(t, tempDir)
	temp2 := openTemp(t, tempDir)

	writeString(t, temp1, "testlog1\n")
	writeString(t, temp2, "testlog2\n")

	require.NoError(t, source.Start())
	defer source.Stop()

	waitForMessages(t, logReceived, []string{"testlog1", "testlog2"})
}

func TestFileSource_MultiFileParallel_PreloadedFiles(t *testing.T) {
	t.Parallel()

	getMessage := func(f, m int) string { return fmt.Sprintf("file %d, message %d", f, m) }

	source, logReceived, tempDir := newTestFileSource(t, nil)

	numFiles := 10
	numMessages := 100

	expected := make([]string, 0, numFiles*numMessages)
	for i := 0; i < numFiles; i++ {
		for j := 0; j < numMessages; j++ {
			expected = append(expected, getMessage(i, j))
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < numFiles; i++ {
		temp := openTemp(t, tempDir)
		wg.Add(1)
		go func(tf *os.File, f int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				writeString(t, tf, getMessage(f, j)+"\n")
			}
		}(temp, i)
	}

	require.NoError(t, source.Start())
	defer source.Stop()

	waitForMessages(t, logReceived, expected)
	wg.Wait()
}

func TestFileSource_MultiFileParallel_LiveFiles(t *testing.T) {
	t.Parallel()

	getMessage := func(f, m int) string { return fmt.Sprintf("file %d, message %d", f, m) }

	source, logReceived, tempDir := newTestFileSource(t, nil)

	numFiles := 10
	numMessages := 100

	expected := make([]string, 0, numFiles*numMessages)
	for i := 0; i < numFiles; i++ {
		for j := 0; j < numMessages; j++ {
			expected = append(expected, getMessage(i, j))
		}
	}

	require.NoError(t, source.Start())
	defer source.Stop()

	temps := make([]*os.File, 0, numFiles)
	for i := 0; i < numFiles; i++ {
		temps = append(temps, openTemp(t, tempDir))
	}

	var wg sync.WaitGroup
	for i, temp := range temps {
		wg.Add(1)
		go func(tf *os.File, f int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				writeString(t, tf, getMessage(f, j)+"\n")
			}
		}(temp, i)
	}

	waitForMessages(t, logReceived, expected)
	wg.Wait()
}

func TestFileSource_MultiFileRotate(t *testing.T) {
	t.Parallel()

	getMessage := func(f, k, m int) string { return fmt.Sprintf("file %d-%d, message %d", f, k, m) }

	source, logReceived, tempDir := newTestFileSource(t, nil)

	numFiles := 3
	numMessages := 3
	numRotations := 3

	expected := make([]string, 0, numFiles*numMessages*numRotations)
	for i := 0; i < numFiles; i++ {
		for j := 0; j < numMessages; j++ {
			for k := 0; k < numRotations; k++ {
				expected = append(expected, getMessage(i, k, j))
			}
		}
	}

	require.NoError(t, source.Start())
	defer source.Stop()

	temps := make([]*os.File, 0, numFiles)
	for i := 0; i < numFiles; i++ {
		temps = append(temps, openTemp(t, tempDir))
	}

	var wg sync.WaitGroup
	for i, temp := range temps {
		wg.Add(1)
		go func(tf *os.File, f int) {
			defer wg.Done()
			for k := 0; k < numRotations; k++ {
				for j := 0; j < numMessages; j++ {
					writeString(t, tf, getMessage(f, k, j)+"\n")
				}

				require.NoError(t, tf.Close())
				require.NoError(t, os.Rename(tf.Name(), fmt.Sprintf("%s.%d", tf.Name(), k)))
				tf = reopenTemp(t, tf.Name())
			}
		}(temp, i)
	}

	waitForMessages(t, logReceived, expected)
	wg.Wait()
}

func TestFileSource_MultiFileRotateSlow(t *testing.T) {
	t.Parallel()

	source, logReceived, tempDir := newTestFileSource(t, nil)

	getMessage := func(f, k, m int) string { return fmt.Sprintf("file %d-%d, message %d", f, k, m) }
	fileName := func(f, k int) string { return filepath.Join(tempDir, fmt.Sprintf("file%d.rot%d.log", f, k)) }
	baseFileName := func(f int) string { return filepath.Join(tempDir, fmt.Sprintf("file%d.log", f)) }

	numFiles := 3
	numMessages := 3
	numRotations := 3

	expected := make([]string, 0, numFiles*numMessages*numRotations)
	for i := 0; i < numFiles; i++ {
		for j := 0; j < numMessages; j++ {
			for k := 0; k < numRotations; k++ {
				expected = append(expected, getMessage(i, k, j))
			}
		}
	}

	require.NoError(t, source.Start())
	defer source.Stop()

	var wg sync.WaitGroup
	for fileNum := 0; fileNum < numFiles; fileNum++ {
		wg.Add(1)
		go func(fileNum int) {
			defer wg.Done()

			for rotationNum := 0; rotationNum < numRotations; rotationNum++ {
				file := openFile(t, baseFileName(fileNum))
				for messageNum := 0; messageNum < numMessages; messageNum++ {
					writeString(t, file, getMessage(fileNum, rotationNum, messageNum)+"\n")
					time.Sleep(20 * time.Millisecond)
				}

				file.Close()
				require.NoError(t, os.Rename(baseFileName(fileNum), fileName(fileNum, rotationNum)))
			}
		}(fileNum)
	}

	waitForMessages(t, logReceived, expected)
	wg.Wait()
}

func TestFileSource_MoveFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Moving files while open is unsupported on Windows")
	}
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp1 := openTemp(t, tempDir)
	writeString(t, temp1, "testlog1\n")
	temp1.Close()

	source.poll(context.Background())

	waitForMessage(t, logReceived, "testlog1")

	// Wait until all goroutines are finished before renaming
	source.wg.Wait()
	err := os.Rename(temp1.Name(), fmt.Sprintf("%s.2", temp1.Name()))
	require.NoError(t, err)

	source.poll(context.Background())
	expectNoMessages(t, logReceived)
}

// TruncateThenWrite tests that, after a file has been truncated,
// any new writes are picked up
func TestFileSource_TruncateThenWrite(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp1 := openTemp(t, tempDir)
	writeString(t, temp1, "testlog1\ntestlog2\n")

	source.poll(context.Background())

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")

	require.NoError(t, temp1.Truncate(0))
	temp1.Seek(0, 0)

	writeString(t, temp1, "testlog3\n")
	source.poll(context.Background())
	waitForMessage(t, logReceived, "testlog3")
	expectNoMessages(t, logReceived)
}

// CopyTruncateWriteBoth tests that when a file is copied
// with unread logs on the end, then the original is truncated,
// we get the unread logs on the copy as well as any new logs
// written to the truncated file
func TestFileSource_CopyTruncateWriteBoth(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp1 := openTemp(t, tempDir)
	writeString(t, temp1, "testlog1\ntestlog2\n")

	source.poll(context.Background())

	waitForMessage(t, logReceived, "testlog1")
	waitForMessage(t, logReceived, "testlog2")
	source.wg.Wait() // wait for all goroutines to finish

	// Copy the first file to a new file, and add another log
	temp2 := openTemp(t, tempDir)
	_, err := io.Copy(temp2, temp1)
	require.NoError(t, err)

	// Truncate original file
	require.NoError(t, temp1.Truncate(0))
	temp1.Seek(0, 0)

	// Write to original and new file
	writeString(t, temp2, "testlog3\n")
	writeString(t, temp1, "testlog4\n")

	// Expect both messages to come through
	source.poll(context.Background())
	waitForMessages(t, logReceived, []string{"testlog3", "testlog4"})
}

// OffsetsAfterRestart tests that a source is able to load
// its offsets after a restart
func TestFileSource_OffsetsAfterRestart(t *testing.T) {
	t.Parallel()
	// Create a new source
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp1 := openTemp(t, tempDir)
	writeString(t, temp1, "testlog1\n")

	// Start the source and expect a message
	require.NoError(t, source.Start())
	defer source.Stop()
	waitForMessage(t, logReceived, "testlog1")

	// Restart the source. Stop and build a new
	// one to guarantee freshness
	require.NoError(t, source.Stop())
	require.NoError(t, source.Start())

	// Write a new log and expect only that log
	writeString(t, temp1, "testlog2\n")
	waitForMessage(t, logReceived, "testlog2")
}

func TestFileSource_OffsetsAfterRestart_BigFiles(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	log1 := stringWithLength(2000)
	log2 := stringWithLength(2000)

	temp1 := openTemp(t, tempDir)
	writeString(t, temp1, log1+"\n")

	// Start the source
	require.NoError(t, source.Start())
	defer source.Stop()
	waitForMessage(t, logReceived, log1)

	// Restart the source
	require.NoError(t, source.Stop())
	require.NoError(t, source.Start())

	writeString(t, temp1, log2+"\n")
	waitForMessage(t, logReceived, log2)
}

func TestFileSource_OffsetsAfterRestart_BigFilesWrittenWhileOff(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	log1 := stringWithLength(2000)
	log2 := stringWithLength(2000)

	temp := openTemp(t, tempDir)
	writeString(t, temp, log1+"\n")

	// Start the source and expect the first message
	require.NoError(t, source.Start())
	defer source.Stop()
	waitForMessage(t, logReceived, log1)

	// Stop the source and write a new message
	require.NoError(t, source.Stop())
	writeString(t, temp, log2+"\n")

	// Start the source and expect the message
	require.NoError(t, source.Start())
	waitForMessage(t, logReceived, log2)
}

func TestFileSource_FileMovedWhileOff_BigFiles(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	log1 := stringWithLength(1000)
	log2 := stringWithLength(1000)

	temp := openTemp(t, tempDir)
	writeString(t, temp, log1+"\n")
	require.NoError(t, temp.Close())

	// Start the source
	require.NoError(t, source.Start())
	defer source.Stop()
	waitForMessage(t, logReceived, log1)

	// Stop the source, then rename and write a new log
	require.NoError(t, source.Stop())

	err := os.Rename(temp.Name(), fmt.Sprintf("%s2", temp.Name()))
	require.NoError(t, err)

	temp = reopenTemp(t, temp.Name())
	require.NoError(t, err)
	writeString(t, temp, log2+"\n")

	// Expect the message written to the new log to come through
	require.NoError(t, source.Start())
	waitForMessage(t, logReceived, log2)
}

func TestFileSource_ManyLogsDelivered(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	count := 1000
	expectedMessages := make([]string, 0, count)
	for i := 0; i < count; i++ {
		expectedMessages = append(expectedMessages, strconv.Itoa(i))
	}

	// Start the source
	require.NoError(t, source.Start())
	defer source.Stop()

	// Write lots of logs
	temp := openTemp(t, tempDir)
	for _, message := range expectedMessages {
		temp.WriteString(message + "\n")
	}

	// Expect each of them to come through once
	for _, message := range expectedMessages {
		waitForMessage(t, logReceived, message)
	}
	expectNoMessages(t, logReceived)
}

func TestFileReader_FingerprintUpdated(t *testing.T) {
	t.Parallel()
	source, logReceived, tempDir := newTestFileSource(t, nil)

	temp := openTemp(t, tempDir)
	tempCopy := openFile(t, temp.Name())
	fp, err := NewFingerprint(temp)
	require.NoError(t, err)
	reader, err := NewReader(temp.Name(), source, tempCopy, fp)
	require.NoError(t, err)

	writeString(t, temp, "testlog1\n")
	reader.LastSeenFileSize = 9
	reader.ReadToEnd(context.Background())
	waitForMessage(t, logReceived, "testlog1")
	require.Equal(t, []byte("testlog1\n"), reader.Fingerprint.FirstBytes)
}

func stringWithLength(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func waitForOne(t *testing.T, c chan *entry.Entry) *entry.Entry {
	select {
	case e := <-c:
		return e
	case <-time.After(time.Minute):
		require.FailNow(t, "Timed out waiting for message")
		return nil
	}
}

func waitForMessage(t *testing.T, c chan *entry.Entry, expected string) {
	select {
	case e := <-c:
		require.Equal(t, expected, e.Record.(string))
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for message", expected)
	}
}

func waitForMessages(t *testing.T, c chan *entry.Entry, expected []string) {
	receivedMessages := make([]string, 0, 100)
LOOP:
	for {
		select {
		case e := <-c:
			receivedMessages = append(receivedMessages, e.Record.(string))
		case <-time.After(500 * time.Millisecond):
			break LOOP
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
			t.Parallel()
			source, receivedEntries, tempDir := newTestFileSource(t, func(cfg *InputConfig) {
				cfg.Encoding = tc.encoding
			})

			// Popualte the file
			temp := openTemp(t, tempDir)
			_, err := temp.Write(tc.contents)
			require.NoError(t, err)

			require.NoError(t, source.Start())
			defer source.Stop()

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

type fileInputBenchmark struct {
	name   string
	config *InputConfig
}

func BenchmarkFileInput(b *testing.B) {
	cases := []fileInputBenchmark{
		{
			"Default",
			NewInputConfig("test_id"),
		},
		{
			"NoFileName",
			func() *InputConfig {
				cfg := NewInputConfig("test_id")
				cfg.IncludeFileName = false
				return cfg
			}(),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			tempDir := testutil.NewTempDir(b)
			path := filepath.Join(tempDir, "in.log")

			cfg := tc.config
			cfg.OutputIDs = []string{"fake"}
			cfg.Include = []string{path}
			cfg.StartAt = "beginning"

			fileOperator, err := cfg.Build(testutil.NewBuildContext(b))
			require.NoError(b, err)

			fakeOutput := testutil.NewFakeOutput(b)
			err = fileOperator.SetOutputs([]operator.Operator{fakeOutput})
			require.NoError(b, err)

			err = fileOperator.Start()
			defer fileOperator.Stop()
			require.NoError(b, err)

			file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
			require.NoError(b, err)

			for i := 0; i < b.N; i++ {
				file.WriteString("testlog\n")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				<-fakeOutput.Received
			}
		})
	}
}
