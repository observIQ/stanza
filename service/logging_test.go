package service

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		conf        func() *LoggingConfig
		expectedErr string
	}{
		{
			name: "empty filename",
			conf: func() *LoggingConfig {
				lc := DefaultLoggingConfig()
				lc.Output = fileOutput
				lc.File.Filename = ""
				return lc
			},
			expectedErr: "file.filename must not be empty",
		},
		{
			name: "empty file struct",
			conf: func() *LoggingConfig {
				lc := DefaultLoggingConfig()
				lc.Output = fileOutput
				lc.File = nil
				return lc
			},
			expectedErr: "'file' key must be specified if file output is specified",
		},
		{
			name: "empty filename doesn't matter for stdout",
			conf: func() *LoggingConfig {
				lc := DefaultLoggingConfig()
				lc.Output = stdOutput
				lc.File.Filename = ""
				return lc
			},
		},
		{
			name: "empty file struct doesn't matter for stdout",
			conf: func() *LoggingConfig {
				lc := DefaultLoggingConfig()
				lc.Output = stdOutput
				lc.File = nil
				return lc
			},
		},
		{
			name: "invalid output",
			conf: func() *LoggingConfig {
				lc := DefaultLoggingConfig()
				lc.Output = "bad_output"
				return lc
			},
			expectedErr: "unknown output type",
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			conf := testCase.conf()
			err := conf.Validate()
			if testCase.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultLoggingConfig(t *testing.T) {
	t.Parallel()

	conf := DefaultLoggingConfig()
	err := conf.Validate()
	require.NoError(t, err)
}

func TestNewFileLogger(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "agent.log")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	config := DefaultLoggingConfig()
	config.Output = fileOutput
	config.File.Filename = file.Name()

	logger := NewLogger(*config)
	logger.Info("test log")

	bytes, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	require.Contains(t, string(bytes), "test log")
}

func TestNewStdLogger(t *testing.T) {
	backup := os.Stdout
	defer func() { os.Stdout = backup }()
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := DefaultLoggingConfig()

	logger := NewLogger(*config)
	logger.Info("test log")

	w.Close()
	results, _ := ioutil.ReadAll(r)
	os.Stdout = backup

	require.Contains(t, string(results), "test log")
}

func TestDefaultEncoder(t *testing.T) {
	t.Parallel()

	encoder := defaultEncoder()
	entry := zapcore.Entry{
		Time: time.Time{},
	}

	buffer, err := encoder.EncodeEntry(entry, nil)
	require.NoError(t, err)

	expected := `{"level":"info","timestamp":"0001-01-01T00:00:00.000Z","message":""}`
	require.Contains(t, string(buffer.Bytes()), expected)
}
