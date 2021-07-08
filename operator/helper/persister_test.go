package helper

import (
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersisterCache(t *testing.T) {
	stubDatabase := database.NewStubDatabase()
	persister := NewScopedDBPersister(stubDatabase, "test")
	persister.Set("key", []byte("value"))
	value := persister.Get("key")
	require.Equal(t, []byte("value"), value)
}

func TestPersisterLoad(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	db, err := database.OpenDatabase(filepath.Join(tempDir, "test.db"))
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err.Error())
		}
	}()
	persister := NewScopedDBPersister(db, "test")
	persister.Set("key", []byte("value"))

	err = persister.Sync()
	require.NoError(t, err)

	newPersister := NewScopedDBPersister(db, "test")
	err = newPersister.Load()
	require.NoError(t, err)

	value := newPersister.Get("key")
	require.Equal(t, []byte("value"), value)
}
