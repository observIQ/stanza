package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/v2/operator/helper/persist"
	"github.com/observiq/stanza/v2/testutil/database"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestOffsetsList(t *testing.T) {
	operatorID1, operatorID2 := "$.testoperatorid1", "$.testoperatorid2"
	tempDir := t.TempDir()

	databasePath := filepath.Join(tempDir, "logagent.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	ioutil.WriteFile(configPath, []byte{}, 0666)

	// capture stdout
	buf := bytes.NewBuffer([]byte{})
	stdout = buf

	// add an offset to the database
	db, err := database.OpenDatabase(databasePath)
	require.NoError(t, err)
	db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(persist.OffsetsBucket)
		require.NoError(t, err)

		err = bucket.Put([]byte(operatorID1), []byte{})
		require.NoError(t, err)
		err = bucket.Put([]byte(operatorID2), []byte{})
		require.NoError(t, err)
		return nil
	})
	db.Close()

	// check that offsets list actually lists the operator
	offsetsList := NewRootCmd()
	offsetsList.SetArgs([]string{
		"offsets", "list",
		"--database", databasePath,
		"--config", configPath,
	})

	expected := fmt.Sprintf("%s\n%s\n", operatorID1, operatorID2)

	err = offsetsList.Execute()
	require.NoError(t, err)
	require.Equal(t, expected, buf.String())

}

func TestOffsetsClear(t *testing.T) {
	operatorID1, operatorID2 := "$.testoperatorid1", "$.testoperatorid2"
	tempDir := t.TempDir()

	databasePath := filepath.Join(tempDir, "logagent.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	ioutil.WriteFile(configPath, []byte{}, 0666)

	// add an offset to the database
	db, err := database.OpenDatabase(databasePath)
	require.NoError(t, err)
	db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(persist.OffsetsBucket)
		require.NoError(t, err)

		err = bucket.Put([]byte(operatorID1), []byte{})
		require.NoError(t, err)
		err = bucket.Put([]byte(operatorID2), []byte{})
		require.NoError(t, err)
		return nil
	})
	db.Close()

	// clear the offsets
	offsetsClear := NewRootCmd()
	offsetsClear.SetArgs([]string{
		"offsets", "clear",
		"--database", databasePath,
		"--config", configPath,
	})

	err = offsetsClear.Execute()
	require.NoError(t, err)
}
