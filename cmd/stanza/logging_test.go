package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestNewFileLogger(t *testing.T) {
	file, err := ioutil.TempFile("", "agent.log")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	flags := RootFlags{
		LogFile: file.Name(),
	}

	logger := newLogger(flags)
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

	flags := RootFlags{
		LogFile: "",
	}

	logger := newLogger(flags)
	logger.Info("test log")

	w.Close()
	results, _ := ioutil.ReadAll(r)
	os.Stdout = backup

	require.Contains(t, string(results), "test log")
}

func TestDefaultEncoder(t *testing.T) {
	encoder := defaultEncoder()
	entry := zapcore.Entry{
		Time: time.Time{},
	}

	buffer, err := encoder.EncodeEntry(entry, nil)
	require.NoError(t, err)

	expected := `{"level":"info","timestamp":"0001-01-01T00:00:00.000Z","message":""}`
	require.Contains(t, string(buffer.Bytes()), expected)
}

func TestGetZapLevel(t *testing.T) {
	testCases := []struct {
		name     string
		level    string
		zapLevel zapcore.Level
	}{
		{
			"Uppercase debug",
			"DEBUG",
			zapcore.DebugLevel,
		},
		{
			"Uppercase info",
			"INFO",
			zapcore.InfoLevel,
		},
		{
			"Uppercase warn",
			"WARN",
			zapcore.WarnLevel,
		},
		{
			"Uppercase error",
			"ERROR",
			zapcore.ErrorLevel,
		},
		{
			"Uppercase panic",
			"PANIC",
			zapcore.PanicLevel,
		},
		{
			"Uppercase fatal",
			"FATAL",
			zapcore.FatalLevel,
		},
		{
			"Lowercase debug",
			"debug",
			zapcore.DebugLevel,
		},
		{
			"Lowercase info",
			"info",
			zapcore.InfoLevel,
		},
		{
			"Lowercase warn",
			"warn",
			zapcore.WarnLevel,
		},
		{
			"Lowercase error",
			"error",
			zapcore.ErrorLevel,
		},
		{
			"Lowercase panic",
			"panic",
			zapcore.PanicLevel,
		},
		{
			"Lowercase fatal",
			"fatal",
			zapcore.FatalLevel,
		},
		{
			"Unknown level",
			"unknown",
			zapcore.InfoLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zapLevel := getZapLevel(tc.level)
			require.Equal(t, tc.zapLevel, zapLevel)
		})
	}
}
