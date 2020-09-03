package buffer

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/semaphore"
)

type MemoryBufferConfig struct {
	MaxEntries int `json:"max_entries" yaml:"max_entries"`
}

func NewMemoryBufferConfig() *MemoryBufferConfig {
	return &MemoryBufferConfig{
		MaxEntries: 1 << 20,
	}
}

func (c MemoryBufferConfig) Build(context operator.BuildContext, pluginID string) (Buffer, error) {
	mb := &MemoryBuffer{
		db:       context.Database,
		pluginID: pluginID,
		buf:      make(chan *entry.Entry, c.MaxEntries),
		sem:      semaphore.NewWeighted(int64(c.MaxEntries)),
		inFlight: make(map[uint64]*entry.Entry, c.MaxEntries),
	}
	if err := mb.loadFromDB(); err != nil {
		return nil, err
	}

	return mb, nil
}

type MemoryBuffer struct {
	db          operator.Database
	pluginID    string
	buf         chan *entry.Entry
	inFlight    map[uint64]*entry.Entry
	inFlightMux sync.Mutex
	entryID     uint64
	sem         *semaphore.Weighted
}

func (m *MemoryBuffer) Add(ctx context.Context, e *entry.Entry) error {
	if err := m.sem.Acquire(ctx, 1); err != nil {
		return err
	}

	select {
	case m.buf <- e:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MemoryBuffer) Read(dst []*entry.Entry) (func(), int, error) {
	inFlight := make([]uint64, len(dst))
	i := 0
	for ; i < len(dst); i++ {
		select {
		case e := <-m.buf:
			dst[i] = e
			id := atomic.AddUint64(&m.entryID, 1)
			m.inFlightMux.Lock()
			m.inFlight[id] = e
			m.inFlightMux.Unlock()
			inFlight[i] = id
		default:
			return m.newFlushFunc(inFlight[:i]), i, nil
		}
	}

	return m.newFlushFunc(inFlight[:i]), i, nil
}

func (m *MemoryBuffer) ReadWait(ctx context.Context, dst []*entry.Entry) (func(), int, error) {
	inFlightIDs := make([]uint64, len(dst))
	i := 0
	for ; i < len(dst); i++ {
		select {
		case e := <-m.buf:
			dst[i] = e
			id := atomic.AddUint64(&m.entryID, 1)
			m.inFlightMux.Lock()
			m.inFlight[id] = e
			m.inFlightMux.Unlock()
			inFlightIDs[i] = id
		case <-ctx.Done():
			return m.newFlushFunc(inFlightIDs[:i]), i, nil
		}
	}

	return m.newFlushFunc(inFlightIDs[:i]), i, nil
}

func (m *MemoryBuffer) newFlushFunc(ids []uint64) func() {
	return func() {
		m.inFlightMux.Lock()
		for _, id := range ids {
			delete(m.inFlight, id)
		}
		m.inFlightMux.Unlock()
		m.sem.Release(int64(len(ids)))
	}
}

func (m *MemoryBuffer) Close() error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		memBufBucket, err := tx.CreateBucketIfNotExists([]byte("memory_buffer"))
		if err != nil {
			return err
		}

		b, err := memBufBucket.CreateBucketIfNotExists([]byte(m.pluginID))
		if err != nil {
			return err
		}

		for k, v := range m.inFlight {
			if err := putKeyValue(b, k, v); err != nil {
				return err
			}
		}

		for {
			select {
			case e := <-m.buf:
				m.entryID++
				if err := putKeyValue(b, m.entryID, e); err != nil {
					return err
				}
			default:
				return nil
			}
		}
	})
}

func putKeyValue(b *bbolt.Bucket, k uint64, v *entry.Entry) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	key := [8]byte{}

	binary.LittleEndian.PutUint64(key[:], k)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return b.Put(key[:], buf.Bytes())
}

func (m *MemoryBuffer) loadFromDB() error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		memBufBucket := tx.Bucket([]byte("memory_buffer"))
		if memBufBucket == nil {
			return nil
		}

		b := memBufBucket.Bucket([]byte(m.pluginID))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			if ok := m.sem.TryAcquire(1); !ok {
				return fmt.Errorf("max_entries is smaller than the number of entries stored in the database")
			}

			dec := json.NewDecoder(bytes.NewReader(v))
			var e entry.Entry
			if err := dec.Decode(&e); err != nil {
				return err
			}

			select {
			case m.buf <- &e:
				return nil
			default:
				return fmt.Errorf("max_entries is smaller than the number of entries stored in the database")
			}
		})
	})
}
