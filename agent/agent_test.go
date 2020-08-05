package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLogAgent(t *testing.T) {
	mockCfg := Config{}
	mockLogger := zap.NewNop().Sugar()
	mockPluginDir := "/some/path/plugins"
	mockDatabaseFile := "/some/path/database"
	mockParameterKey := "test"
	mockParameterValue := "value"
	agent := NewLogAgent(&mockCfg, mockLogger, mockPluginDir, mockDatabaseFile).
		WithBuildParameter(mockParameterKey, mockParameterValue)

	require.Equal(t, &mockCfg, agent.Config)
	require.Equal(t, mockLogger, agent.SugaredLogger)
	require.Equal(t, mockPluginDir, agent.PluginDir)
	require.Equal(t, mockDatabaseFile, agent.Database)
	require.Equal(t, mockParameterValue, agent.buildParams[mockParameterKey])
}

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
