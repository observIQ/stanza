package fileinput

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"go.etcd.io/bbolt"
)

type OffsetStore struct {
	db     *bbolt.DB
	bucket string
}

type offsetStorer struct {
	Offset int64
}

func (s *OffsetStore) GetOffset(fingerprint []byte) (*int64, error) {
	var offset *int64
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(s.bucket))
		if bucket == nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}
		val := bucket.Get(fingerprint)
		if val == nil {
			return nil
		}

		var storer offsetStorer
		err = gob.NewDecoder(bytes.NewReader(val)).Decode(&storer)
		if err != nil {
			return err
		}

		offset = &storer.Offset
		return nil
	})

	return offset, err
}

func (s *OffsetStore) SetOffset(fingerprint []byte, offset int64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(s.bucket))
		if bucket == nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}

		var buf bytes.Buffer
		storer := offsetStorer{
			Offset: offset,
		}
		err = gob.NewEncoder(&buf).Encode(storer)
		if err != nil {
			return err
		}

		return bucket.Put(fingerprint, buf.Bytes())
	})
}
