package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/observiq/bplogagent/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpenDatabase(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		tempDir := testutil.NewTempDir(t)
		db, err := OpenDatabase(filepath.Join(tempDir, "test.db"))
		require.NoError(t, err)
		require.NotNil(t, db)
	})

	t.Run("NonexistantPathIsCreated", func(t *testing.T) {
		tempDir := testutil.NewTempDir(t)
		db, err := OpenDatabase(filepath.Join(tempDir, "nonexistdir", "test.db"))
		require.NoError(t, err)
		require.NotNil(t, db)
	})

	t.Run("BadPermissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not have the same kind of file permissions")
		}
		tempDir := testutil.NewTempDir(t)
		err := os.MkdirAll(filepath.Join(tempDir, "badperms"), 0666)
		require.NoError(t, err)
		db, err := OpenDatabase(filepath.Join(tempDir, "badperms", "nonexistdir", "test.db"))
		require.Error(t, err)
		require.Nil(t, db)
	})
}
