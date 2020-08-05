package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/observiq/carbon/operator"
	"go.etcd.io/bbolt"
	"go.uber.org/zap/zaptest"
)

// NewTempDir will return a new temp directory for testing
func NewTempDir(t TestingT) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// NewTestDatabase will return a new database for testing
func NewTestDatabase(t TestingT) *bbolt.DB {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	db, err := bbolt.Open(filepath.Join(tempDir, "test.db"), 0666, nil)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// NewBuildContext will return a new build context for testing
func NewBuildContext(t TestingT) operator.BuildContext {
	return operator.BuildContext{
		PluginRegistry: make(operator.PluginRegistry),
		Database:       NewTestDatabase(t),
		Logger:         zaptest.NewLogger(t).Sugar(),
	}
}

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

type TestingT interface {
	// Logs the given message without failing the test.
	Logf(string, ...interface{})

	// Logs the given message and marks the test as failed.
	Errorf(string, ...interface{})

	// Marks the test as failed.
	Fail()

	// Returns true if the test has been marked as failed.
	Failed() bool

	// Returns the name of the test.
	Name() string

	// Marks the test as failed and stops execution of that test.
	FailNow()

	Cleanup(func())
}
