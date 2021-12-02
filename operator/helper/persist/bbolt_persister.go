package persist

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

// PersisterShutdownFunc handles cleanup of the persister
type PersisterShutdownFunc func() error

// OffsetsBucket is the scope provided to offset persistence
var OffsetsBucket = []byte(`offsets`)

// BBoltPersister is a persister that uses a database for the backend
type BBoltPersister struct {
	db *bbolt.DB
}

// NewBBoltPersister opens a connection to a bbolt database for the given filePath and wraps it in a Scopped Persister.
// scope refers to a bbolt bucket name
func NewBBoltPersister(filePath string) (*BBoltPersister, PersisterShutdownFunc, error) {
	// Verify directory exists for bbolt
	if _, err := os.Stat(filepath.Dir(filePath)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err := os.MkdirAll(filepath.Dir(filePath), 0755) // #nosec - 0755 directory permissions are okay
			if err != nil {
				return nil, nil, fmt.Errorf("creating database directory: %w", err)
			}
		} else {
			return nil, nil, err
		}
	}

	// Open connection to bbolt
	options := &bbolt.Options{Timeout: 1 * time.Second}
	db, err := bbolt.Open(filePath, 0600, options)
	if err != nil {
		return nil, nil, fmt.Errorf("error while opening bbolt connection: %w", err)
	}

	persister := &BBoltPersister{
		db: db,
	}

	shutDownFunc := func() error {
		return db.Close()
	}
	return persister, shutDownFunc, nil
}

// Get returns the data associated with the key
func (p *BBoltPersister) Get(_ context.Context, key string) ([]byte, error) {
	var value []byte

	updateErr := p.db.Update(func(tx *bbolt.Tx) error {

		bucket, err := tx.CreateBucketIfNotExists(OffsetsBucket)
		if err != nil {
			return err
		}

		value = bucket.Get([]byte(key))
		return nil
	})

	return value, updateErr
}

// Set sets the data associated with the key to the bbolt scoped bucket
func (p *BBoltPersister) Set(_ context.Context, key string, value []byte) error {
	return p.db.Update(func(tx *bbolt.Tx) error {

		bucket, err := tx.CreateBucketIfNotExists(OffsetsBucket)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), value)
	})
}

// Delete removes the key from the bbolt scoped bucket
func (p *BBoltPersister) Delete(_ context.Context, key string) error {
	return p.db.Update(func(tx *bbolt.Tx) error {

		bucket, err := tx.CreateBucketIfNotExists(OffsetsBucket)
		if err != nil {
			return err
		}

		return bucket.Delete([]byte(key))
	})
}

// Clear clears the offset bucket of all data
func (p *BBoltPersister) Clear() error {
	err := p.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(OffsetsBucket)
		if bucket != nil {
			return tx.DeleteBucket(OffsetsBucket)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error while clearing bucket: %w", err)
	}

	return p.db.Sync()
}

// Keys returns all the keys associated with the bbolt db backing this persister
func (p *BBoltPersister) Keys() ([]string, error) {
	keys := make([]string, 0)

	err := p.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(OffsetsBucket)
		return bucket.ForEach(func(key, value []byte) error {
			keys = append(keys, string(key))

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return keys, nil
}
