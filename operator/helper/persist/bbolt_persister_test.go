package persist

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBBoltPersister(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Creates directory if not exists",
			testFunc: func(t *testing.T) {
				tempDir := t.TempDir()
				dirPath := filepath.Join(tempDir, "not_existing")
				filePath := filepath.Join(dirPath, "my_db")
				db, err := NewBBoltPersister(filePath)
				require.NoError(t, err)
				require.NotNil(t, db)
				require.DirExists(t, dirPath)
				require.FileExists(t, filePath)

				// Windows file permissions will match the unix ones
				if runtime.GOOS != "windows" {
					info, err := os.Stat(dirPath)
					require.NoError(t, err)
					assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
				}
			},
		},
		{
			desc: "Dir Permissions issues",
			testFunc: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not have the same kind of file permissions")
				}

				tempDir := t.TempDir()
				dirPath := filepath.Join(tempDir, "badperms")
				err := os.MkdirAll(dirPath, 0111)
				require.NoError(t, err)

				filePath := filepath.Join(dirPath, "db_dir", "my_db")
				db, err := NewBBoltPersister(filePath)
				require.Error(t, err)
				require.Nil(t, db)
			},
		},
		{
			desc: "File Permissions issues",
			testFunc: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not have the same kind of file permissions")
				}

				tempDir := t.TempDir()

				filePath := filepath.Join(tempDir, "my_db.db")
				_, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0111)
				require.NoError(t, err)
				db, err := NewBBoltPersister(filePath)
				require.Error(t, err)
				require.Nil(t, db)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
