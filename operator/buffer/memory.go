package buffer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/stanza/entry"
	"golang.org/x/sync/semaphore"
)

type MemoryBufferConfig struct {
	MaxEvents int `json:"max_events" yaml:"max_events"`
}

func (c MemoryBufferConfig) Build() Buffer {
	return NewMemoryBuffer(c.MaxEvents)
}

type MemoryBuffer struct {
	// TODO flush to database?
	buf         chan *entry.Entry
	inFlight    map[int64]*entry.Entry
	inFlightMux sync.Mutex
	entryID     int64
	sem         *semaphore.Weighted
}

func NewMemoryBuffer(size int) *MemoryBuffer {
	return &MemoryBuffer{
		buf:      make(chan *entry.Entry, size),
		sem:      semaphore.NewWeighted(int64(size)),
		inFlight: make(map[int64]*entry.Entry, size),
	}
}

func (m *MemoryBuffer) Add(ctx context.Context, e *entry.Entry) error {
	if err := m.sem.Acquire(ctx, 1); err != nil {
		return err
	}

	select {
	case m.buf <- e:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

func (m *MemoryBuffer) Read(dst []*entry.Entry) (func(), int, error) {
	inFlight := make([]int64, len(dst))
	i := 0
	for ; i < len(dst); i++ {
		select {
		case e := <-m.buf:
			dst[i] = e
			id := atomic.AddInt64(&m.entryID, 1)
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

func (m *MemoryBuffer) ReadWait(dst []*entry.Entry, timeout <-chan time.Time) (func(), int, error) {
	inFlight := make([]int64, len(dst))
	i := 0
	for ; i < len(dst); i++ {
		select {
		case e := <-m.buf:
			dst[i] = e
			id := atomic.AddInt64(&m.entryID, 1)
			m.inFlightMux.Lock()
			m.inFlight[id] = e
			m.inFlightMux.Unlock()
			inFlight[i] = id
		case <-timeout:
			return m.newFlushFunc(inFlight[:i]), i, nil
		}
	}

	return m.newFlushFunc(inFlight[:i]), i, nil
}

func (m *MemoryBuffer) newFlushFunc(ids []int64) func() {
	return func() {
		m.inFlightMux.Lock()
		for _, id := range ids {
			delete(m.inFlight, id)
		}
		m.inFlightMux.Unlock()
		m.sem.Release(int64(len(ids)))
	}
}
