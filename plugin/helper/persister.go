package helper

import (
	"sync"

	"github.com/observiq/bplogagent/plugin"
	"go.etcd.io/bbolt"
)

type Persister interface {
	Get(string) []byte
	Set(string, []byte)
	Sync() error
	Load() error
}

type ScopedBBoltPersister struct {
	scope    []byte
	db       plugin.Database
	cache    map[string][]byte
	cacheMux sync.Mutex
}

func NewScopedDBPersister(db plugin.Database, scope string) *ScopedBBoltPersister {
	return &ScopedBBoltPersister{
		scope: []byte(scope),
		db:    db,
		cache: make(map[string][]byte),
	}
}

// Set retrieves a key from the cache
func (p *ScopedBBoltPersister) Get(key string) []byte {
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()
	return p.cache[key]
}

// Set saves a key in the cache
func (p *ScopedBBoltPersister) Set(key string, val []byte) {
	p.cacheMux.Lock()
	p.cache[key] = val
	p.cacheMux.Unlock()
}

var OffsetsBucket = []byte(`offsets`)

// Sync saves the cache to the backend, ensuring values are
// safely written to disk before returning
func (p *ScopedBBoltPersister) Sync() error {
	return p.db.Update(func(tx *bbolt.Tx) error {
		offsetBucket, err := tx.CreateBucketIfNotExists(OffsetsBucket)
		if err != nil {
			return err
		}

		bucket, err := offsetBucket.CreateBucketIfNotExists(p.scope)
		if err != nil {
			return err
		}

		p.cacheMux.Lock()
		for k, v := range p.cache {
			err := bucket.Put([]byte(k), v)
			if err != nil {
				return err
			}
		}
		p.cacheMux.Unlock()

		return nil
	})
}

// Load populates the cache with the values from the database,
// overwriting anything currently in the cache.
func (p *ScopedBBoltPersister) Load() error {
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()
	p.cache = make(map[string][]byte)

	return p.db.Update(func(tx *bbolt.Tx) error {
		offsetBucket, err := tx.CreateBucketIfNotExists(OffsetsBucket)
		if err != nil {
			return err
		}

		bucket, err := offsetBucket.CreateBucketIfNotExists(p.scope)
		if err != nil {
			return err
		}

		return bucket.ForEach(func(k, v []byte) error {
			p.cache[string(k)] = v
			return nil
		})
	})
}
