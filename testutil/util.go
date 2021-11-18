package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observiq/stanza/v2/logger"
	"github.com/observiq/stanza/v2/operator"
	"go.etcd.io/bbolt"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// NewTempDir will return a new temp directory for testing
func NewTempDir(t testing.TB) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf(err.Error())
		}
	})

	return tempDir
}

// NewTestDatabase will return a new database for testing
func NewTestDatabase(t testing.TB) *bbolt.DB {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf(err.Error())
		}
	})

	db, err := bbolt.Open(filepath.Join(tempDir, "test.db"), 0666, nil)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf(err.Error())
		}
	})

	return db
}

// NewBuildContext will return a new build context for testing
func NewBuildContext(t testing.TB) operator.BuildContext {
	return operator.BuildContext{
		Database:  NewTestDatabase(t),
		Logger:    logger.New(zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel)).Sugar()),
		Namespace: "$",
	}
}

// Trim removes white space from the lines of a string
func Trim(s string) string {
	lines := strings.Split(s, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		trimmed = append(trimmed, strings.Trim(line, " \t\n"))
	}

	return strings.Join(trimmed, "\n")
}
