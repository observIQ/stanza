package main

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
		conf        func() loggingConfig
		expectedErr string
	}{
		{
			name: "empty filename",
			conf: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = fileOutput
				lc.File.Filename = ""
				return lc
			},
			expectedErr: "file.filename must not be empty",
		},
		{
			name: "empty file struct",
			conf: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = fileOutput
				lc.File = nil
				return lc
			},
			expectedErr: "'file' key must be specified if file output is specified",
		},
		{
			name: "empty filename doesn't matter for stdout",
			conf: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = stdOutput
				lc.File.Filename = ""
				return lc
			},
			expectedErr: "",
		},
		{
			name: "empty file struct doesn't matter for stdout",
			conf: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = stdOutput
				lc.File = nil
				return lc
			},
			expectedErr: "",
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			conf := testCase.conf()
			err := conf.validate()
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

	conf := defaultLoggingConfig()
	err := conf.validate()
	require.NoError(t, err)
}

func TestGetLoggingConfig(t *testing.T) {
	testCases := []struct {
		name        string
		yaml        string
		expected    func() loggingConfig
		expectedErr string
	}{
		{
			name:     "empty logging config",
			yaml:     ``,
			expected: defaultLoggingConfig,
		},
		{
			name:     "stdout logging config",
			yaml:     `output: stdout`,
			expected: defaultLoggingConfig,
		},
		{
			name: "file logging config",
			yaml: `output: file`,
			expected: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = fileOutput
				return lc
			},
		},
		{
			name: "file logging config options",
			yaml: `
output: file
file:
  filename: "example.log"
  maxbackups: 15
  maxsize: 16
  maxage: 17
`,
			expected: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Output = fileOutput
				lc.File.Filename = "example.log"
				lc.File.MaxBackups = 15
				lc.File.MaxSize = 16
				lc.File.MaxAge = 17
				return lc
			},
		},
		{
			name: "info log level",
			yaml: `
output: stdout
level: info
`,
			expected: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Level = zapcore.InfoLevel
				return lc
			},
		},
		{
			name: "debug log level",
			yaml: `
output: stdout
level: debug
`,
			expected: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Level = zapcore.DebugLevel
				return lc
			},
		},
		{
			name: "error log level",
			yaml: `
output: stdout
level: error
`,
			expected: func() loggingConfig {
				lc := defaultLoggingConfig()
				lc.Level = zapcore.ErrorLevel
				return lc
			},
		},
		{
			name: "unknown field in yaml",
			yaml: `
some_field: a
`,
			expectedErr: "field some_field not found",
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f, err := ioutil.TempFile("", "test-logging-config")
			require.NoError(t, err)
			path := f.Name()
			defer os.Remove(path)

			_, err = f.Write([]byte(testCase.yaml))
			require.NoError(t, err)

			err = f.Close()
			require.NoError(t, err)

			conf, err := getLoggingConfig(&RootFlags{
				LogConfig: path,
			})

			if testCase.expectedErr == "" {
				require.NoError(t, err)
				require.Equal(t, testCase.expected(), conf)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.expectedErr)
			}
		})
	}
}

func TestGetLoggingConfigNoFile(t *testing.T) {
	t.Parallel()

	conf, err := getLoggingConfig(&RootFlags{})
	require.NoError(t, err)
	require.Equal(t, defaultLoggingConfig(), conf)
}

func TestNewFileLogger(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "agent.log")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	config := defaultLoggingConfig()
	config.Output = fileOutput
	config.File.Filename = file.Name()

	logger := newLogger(config)
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

	config := defaultLoggingConfig()

	logger := newLogger(config)
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
