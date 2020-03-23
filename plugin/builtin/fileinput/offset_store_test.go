package fileinput

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func TestOffsetStoreRoundtrip(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tempDir)
		assert.NoError(t, err)
	}()

	db, err := bbolt.Open(filepath.Join(tempDir, "bplogagent.db"), 0666, nil)
	assert.NoError(t, err)

	store := &OffsetStore{
		db:     db,
		bucket: "test",
	}

	offset, err := store.GetOffset([]byte(`asdfasdf`))
	assert.NoError(t, err)
	assert.Nil(t, offset)

	err = store.SetOffset([]byte(`asdfasdf`), 123)
	assert.NoError(t, err)

	offset, err = store.GetOffset([]byte(`asdfasdf`))
	assert.NoError(t, err)
	assert.NotNil(t, offset)
	assert.Equal(t, int64(123), *offset)

}
