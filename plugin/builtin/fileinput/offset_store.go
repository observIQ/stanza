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
	offset int64
}

func (s *OffsetStore) GetOffset(fingerprint []byte) (*int64, error) {
	var offset *int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(s.bucket))
		if bucket == nil {
			return fmt.Errorf("bucket '%s' does not exist", s.bucket)
		}
		val := bucket.Get(fingerprint)
		if val == nil {
			offset = func() *int64 { i := int64(0); return &i }()
			return nil
		}

		var storer offsetStorer
		err := gob.NewDecoder(bytes.NewReader(val)).Decode(&storer)
		if err != nil {
			return err
		}

		offset = &storer.offset
		return nil
	})

	return offset, err
}

func (s *OffsetStore) SetOffset(fingerprint []byte, offset int64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(s.bucket))
		if bucket == nil {
			return fmt.Errorf("bucket '%s' does not exist", s.bucket)
		}

		buf := make([]byte, 0, 10)
		storer := offsetStorer{
			offset: offset,
		}
		err := gob.NewEncoder(bytes.NewBuffer(buf)).Encode(storer)
		if err != nil {
			return err
		}

		return bucket.Put(fingerprint, buf)
	})
}
