package commands

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	agent "github.com/bluemedora/bplogagent/agent"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestOffsets(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	databasePath := filepath.Join(tempDir, "logagent.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	ioutil.WriteFile(configPath, []byte{}, 0666)

	// capture stdout
	buf := bytes.NewBuffer([]byte{})
	stdout = buf

	// add an offset to the database
	db, err := agent.OpenDatabase(databasePath)
	require.NoError(t, err)
	db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(helper.OffsetsBucket)
		require.NoError(t, err)

		_, err = bucket.CreateBucket([]byte("$.testpluginid1"))
		require.NoError(t, err)
		_, err = bucket.CreateBucket([]byte("$.testpluginid2"))
		require.NoError(t, err)
		return nil
	})
	db.Close()

	// check that offsets list actually lists the plugin
	offsetsList := NewRootCmd()
	offsetsList.SetArgs([]string{
		"offsets", "list",
		"--database", databasePath,
		"--config", configPath,
	})

	err = offsetsList.Execute()
	require.NoError(t, err)
	require.Equal(t, "$.testpluginid1\n$.testpluginid2\n", buf.String())

	// clear the offsets
	offsetsClear := NewRootCmd()
	offsetsClear.SetArgs([]string{
		"offsets", "clear",
		"--database", databasePath,
		"--config", configPath,
		"$.testpluginid2",
	})

	err = offsetsClear.Execute()
	require.NoError(t, err)

	// Check that offsets list only shows uncleared plugin id
	buf.Reset()
	err = offsetsList.Execute()
	require.NoError(t, err)
	require.Equal(t, "$.testpluginid1\n", buf.String())

}
