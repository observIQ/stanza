package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/carbon/plugin"
	"go.etcd.io/bbolt"
	"go.uber.org/zap/zaptest"
)

func NewTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

func NewTestDatabase(t *testing.T) *bbolt.DB {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	db, err := bbolt.Open(filepath.Join(tempDir, "test.db"), 0666, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func NewBuildContext(t *testing.T) plugin.BuildContext {
	return plugin.BuildContext{
		Database: NewTestDatabase(t),
		Logger:   zaptest.NewLogger(t).Sugar(),
	}
}
