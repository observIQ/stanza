package persist

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
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
				persister, shutdownFunc, err := NewBBoltPersister(filePath)
				require.NoError(t, err)
				defer shutdownFunc()
				require.NotNil(t, persister)
				require.DirExists(t, dirPath)
				require.FileExists(t, filePath)

				// Windows file permissions will not match the unix ones
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
				persister, _, err := NewBBoltPersister(filePath)
				require.Error(t, err)
				require.Nil(t, persister)
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
				persister, _, err := NewBBoltPersister(filePath)
				require.Error(t, err)
				require.Nil(t, persister)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestBBoltPersisterGet(t *testing.T) {
	seedKey, seedData := "Seed", []byte("data")
	testCases := []struct {
		desc         string
		key          string
		expectedData []byte
		expectError  bool
	}{
		{
			desc:         "Retrieve Seed Data",
			key:          seedKey,
			expectedData: seedData,
			expectError:  false,
		},
		{
			desc:         "Key not found",
			key:          "badKey",
			expectedData: nil,
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup persister
			filePath := filepath.Join(tempDir, "my_db.db")
			persister, shutdownFunc, err := NewBBoltPersister(filePath)
			defer shutdownFunc()
			require.NoError(t, err)

			// Add seed Data
			require.NoError(t, persister.Set(context.Background(), seedKey, seedData))

			// Get Data
			data, err := persister.Get(context.Background(), tc.key)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectedData, data)
		})
	}
}

func TestBBoltPersisterSet(t *testing.T) {

	testCases := []struct {
		desc        string
		key         string
		data        []byte
		expectError bool
	}{
		{
			desc:        "Successful Set",
			key:         "key",
			data:        []byte("data"),
			expectError: false,
		},
		{
			desc:        "Blank Key",
			key:         "",
			data:        []byte("data"),
			expectError: true,
		},
		{
			desc:        "Key to large",
			key:         string(make([]byte, bbolt.MaxKeySize+1)),
			data:        []byte("data"),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup persister
			filePath := filepath.Join(tempDir, "my_db.db")
			persister, shutdownFunc, err := NewBBoltPersister(filePath)
			defer shutdownFunc()
			require.NoError(t, err)

			// Set data
			err = persister.Set(context.Background(), tc.key, tc.data)
			if tc.expectError {
				require.Error(t, err)
				// We expect and error therefore the following Get operation will fail so just return from this test func
				return
			}

			// Expect no set error
			require.NoError(t, err)

			// Verify set data is the same that we retrieve
			data, err := persister.Get(context.Background(), tc.key)
			require.NoError(t, err)
			assert.Equal(t, tc.data, data)
		})
	}
}

// Note on this test because of how the bucket name and transactions are currently hard coded it's not possible
// to hit the error cases of tx.Delete. This can only happen if the bucket is a readonly one and we create a writable bucket.
func TestBBoltPersisterDelete(t *testing.T) {
	seedKey, seedData := "Seed", []byte("data")
	testCases := []struct {
		desc         string
		key          string
		expectDelete bool
		expectError  bool
	}{
		{
			desc:         "Successful Delete",
			key:          seedKey,
			expectDelete: true,
			expectError:  false,
		},
		{
			desc:         "Delete called on non-existing key",
			key:          "not a key",
			expectDelete: false,
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup persister
			filePath := filepath.Join(tempDir, "my_db.db")
			persister, shutdownFunc, err := NewBBoltPersister(filePath)
			defer shutdownFunc()
			require.NoError(t, err)

			// Add seed Data
			require.NoError(t, persister.Set(context.Background(), seedKey, seedData))

			// Expect no set error
			err = persister.Delete(context.Background(), tc.key)
			if tc.expectError {
				require.Error(t, err)
				// We expect and error therefore the following Get operation will fail so just return from this test func
				return
			}

			// Expect no delete error
			require.NoError(t, err)

			// If we expect the seed data to be gone verify it is
			data, err := persister.Get(context.Background(), seedKey)
			require.NoError(t, err)
			if tc.expectDelete {
				assert.Nil(t, data)
			} else {
				assert.NotNil(t, data)
			}
		})
	}
}

func TestBBoltPersisterClear(t *testing.T) {
	tempDir := t.TempDir()

	// Setup persister
	filePath := filepath.Join(tempDir, "my_db.db")
	persister, shutdownFunc, err := NewBBoltPersister(filePath)
	defer shutdownFunc()
	require.NoError(t, err)

	// Store keys for later lookup
	keys := make([]string, 10)

	// Set seed data
	for i := 0; i < len(keys); i++ {
		key := strconv.Itoa(i)
		keys[i] = key
		data := []byte(key + "data")
		err := persister.Set(context.Background(), key, data)
		require.NoError(t, err)
	}

	// Clear the persister
	// NOTE: Error case is not possible to hit (unless the os file descripter write fails) due to using a hard coded value for our bucket
	err = persister.Clear()
	require.NoError(t, err)

	// Assert all keys have been removed
	for _, key := range keys {
		data, err := persister.Get(context.Background(), key)
		require.NoError(t, err)
		assert.Nil(t, data)
	}
}

func TestBBoltPersisterKeys(t *testing.T) {
	tempDir := t.TempDir()

	// Setup persister
	filePath := filepath.Join(tempDir, "my_db.db")
	persister, shutdownFunc, err := NewBBoltPersister(filePath)
	defer shutdownFunc()
	require.NoError(t, err)

	// Store expectedKeys for later lookup
	expectedKeys := make([]string, 10)

	// Set seed data
	for i := 0; i < len(expectedKeys); i++ {
		key := strconv.Itoa(i)
		expectedKeys[i] = key
		data := []byte(key + "data")
		err := persister.Set(context.Background(), key, data)
		require.NoError(t, err)
	}

	keys, err := persister.Keys()
	require.NoError(t, err)

	// Sort both slices so they are easier to compare
	sort.Slice(expectedKeys, func(i, j int) bool {
		return expectedKeys[i] < expectedKeys[j]
	})

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	assert.Equal(t, expectedKeys, keys)

}
