package plugin

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"go.etcd.io/bbolt"
)

type DiskBufferConfig struct {
}

type DiskBuffer struct {
	BaseBuffer `mapstructure:",squash"`

	bucket         string
	db             *bbolt.DB
	bundleFlushers sync.Map
}

func (b *DiskBuffer) Input(entry *entry.Entry) error {
	buf, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return b.db.Update(func(tx *bbolt.Tx) error {
		// Get the current bundle
		bucket := tx.Bucket([]byte(b.bucket))
		// TODO nil safety checks
		bundleID := bucket.Get([]byte("activeBundleID"))
		bundle := bucket.Bucket(bundleID)

		// If the current bundle is full, mark the next bundle as active
		bundleStats := bundle.Stats()
		count := bundleStats.KeyN
		bytes := bundleStats.LeafInuse
		if count > b.bundleCountLimit || bytes+len(buf) > b.bundleByteLimit {
			// Ensure the bundle is flushed
			b.flushBundleOnce(bundleID)

			// Create and use a new bundle
			next, _ := bucket.NextSequence()
			bundleID = itob(next)
			bundle, _ = bucket.CreateBucket(bundleID)
			_ = bucket.Put([]byte("activeBundleID"), bundleID) // TODO handle error?

			// Flush the bundle after the configured delay
			go func() {
				// TODO cancel this with a context
				<-time.After(time.Duration(float64(time.Second) * b.flushDelayThreshold))
				b.flushBundleOnce(bundleID)
			}()
		}

		// Insert the log into the active bundle
		entryID, _ := bundle.NextSequence()
		err := bundle.Put(itob(entryID), buf)
		if err != nil {
			return err
		}

		// If this log puts the bundle over its flush threshold, mark it as flushable
		if count > b.bundleCountThreshold || bytes+len(buf) > b.bundleByteThreshold {
			b.flushBundleOnce(bundleID)
		}

		return nil
	})
}

// flushBundleOnce starts a flush goroutine if one hasn't been started for bundleID
func (b *DiskBuffer) flushBundleOnce(bundleID []byte) {
	// Ensure that we only flush the bundle once by atomically loading a map
	_, ok := b.bundleFlushers.LoadOrStore(bundleID, struct{}{})
	if !ok {
		go func() {
			// TODO provide a cancellable context
			b.flushBundleWithRetry(bundleID)
		}()
	}
}

// flushBundleWithRetry continues attempting to flush a bundle until it succeeds.
// When it succeeds, it deletes the flushed bundle from the database.
func (b *DiskBuffer) flushBundleWithRetry(bundleID []byte) {
	// TODO limit concurrent flushes

	entries, err := b.getBundleEntries(bundleID)
	if err != nil {
		// TODO log an error
	}

	for {
		// TODO retry with backoff
		err := b.handler(entries)
		if err != nil {
			// TODO log
			continue
		}

		break
	}

	b.deleteBundle(bundleID)
}

// deleteBundle deletes a bundle from the database
func (b *DiskBuffer) deleteBundle(bundleID []byte) {
	b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucket))
		_ = bucket.DeleteBucket(bundleID) // TODO handle error
		return nil
	})
}

// getBundleEntries returns an array of unmarshalled entries.
// Once we retrieve the entries, that bundle can no longer be written to, otherwise the
// newly written logs will not be processed. We ensure this doesn't happen by creating
// a new active bundle if bundleID is currently the active bundle.
func (b *DiskBuffer) getBundleEntries(bundleID []byte) ([]*entry.Entry, error) {
	var entries []*entry.Entry
	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucket))
		bundle := bucket.Bucket(bundleID)

		// Create a new bundle
		newBundleID, _ := bucket.NextSequence()
		bundle, _ = bucket.CreateBucket(itob(newBundleID))
		_ = bucket.Put([]byte("activeBundleID"), itob(newBundleID)) // TODO handle error?

		// Flush the bundle after the configured delay
		go func() {
			// TODO cancel this with a context
			<-time.After(time.Duration(float64(time.Second) * b.flushDelayThreshold))
			b.flushBundleOnce(bundleID)
		}()

		// Create the entry list from the marshalled entries
		entries = make([]*entry.Entry, bundle.Stats().LeafInuse)
		return bundle.ForEach(func(_, v []byte) error {
			var newEntry *entry.Entry
			err := json.Unmarshal(v, &newEntry)
			if err != nil {
				// TODO log an error
				return nil
			}
			entries = append(entries, newEntry)
			return nil
		})
	})

	return entries, err
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
