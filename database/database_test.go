package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewTempDir will return a new temp directory for testing
func NewTempDir(t testing.TB) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("%v", err)
		t.FailNow()
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

func TestOpenDatabase(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		tempDir := NewTempDir(t)
		db, err := OpenDatabase(filepath.Join(tempDir, "test.db"))
		require.NoError(t, err)
		require.NotNil(t, db)
	})

	t.Run("NoFile", func(t *testing.T) {
		db, err := OpenDatabase("")
		require.NoError(t, err)
		require.NotNil(t, db)
		require.IsType(t, &StubDatabase{}, db)
	})

	t.Run("NonexistantPathIsCreated", func(t *testing.T) {
		tempDir := NewTempDir(t)
		db, err := OpenDatabase(filepath.Join(tempDir, "nonexistdir", "test.db"))
		require.NoError(t, err)
		require.NotNil(t, db)
	})

	t.Run("BadPermissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not have the same kind of file permissions")
		}
		tempDir := NewTempDir(t)
		err := os.MkdirAll(filepath.Join(tempDir, "badperms"), 0666)
		require.NoError(t, err)
		db, err := OpenDatabase(filepath.Join(tempDir, "badperms", "nonexistdir", "test.db"))
		require.Error(t, err)
		require.Nil(t, db)
	})

	t.Run("ExecuteOnlyPermissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not have the same kind of file permissions")
		}
		tempDir := NewTempDir(t)
		err := os.MkdirAll(filepath.Join(tempDir, "badperms"), 0111)
		require.NoError(t, err)
		db, err := OpenDatabase(filepath.Join(tempDir, "badperms", "nonexistdir", "test.db"))
		require.Error(t, err)
		require.Nil(t, db)
	})

}

func TestStubDatabase(t *testing.T) {
	stubDatabase := NewStubDatabase()
	err := stubDatabase.Close()
	require.NoError(t, err)

	err = stubDatabase.Sync()
	require.NoError(t, err)

	err = stubDatabase.Update(nil)
	require.NoError(t, err)

	err = stubDatabase.View(nil)
	require.NoError(t, err)
}
