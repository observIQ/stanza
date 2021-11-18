package cloudwatch

import (
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/v2/database"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersisterCache(t *testing.T) {
	stubDatabase := database.NewStubDatabase()
	persister := Persister{
		DB: helper.NewScopedDBPersister(stubDatabase, "test"),
	}
	persister.Write("key", int64(1620666055012))
	value, readErr := persister.Read("key")
	require.NoError(t, readErr)
	require.Equal(t, int64(1620666055012), value)
}

func TestPersisterLoad(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	db, openDbErr := database.OpenDatabase(filepath.Join(tempDir, "test.db"))
	require.NoError(t, openDbErr)
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err.Error())
		}
	}()
	persister := Persister{
		DB: helper.NewScopedDBPersister(db, "test"),
	}
	persister.Write("key", 1620666055012)

	syncErr := persister.DB.Sync()
	require.NoError(t, syncErr)

	loadErr := persister.DB.Load()
	require.NoError(t, loadErr)

	value, readErr := persister.Read("key")
	require.NoError(t, readErr)
	require.Equal(t, int64(1620666055012), value)
}
