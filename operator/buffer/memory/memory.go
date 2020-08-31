package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
	"golang.org/x/sync/semaphore"
)

var _ buffer.Buffer = &MemoryBuffer{}

type MemoryBuffer struct {
	buf      chan *entry.Entry
	inFlight sync.Map // TODO benchmark against typed, locked map
	entryID  int64
	sem      *semaphore.Weighted
}

func NewMemoryBuffer(size int64) *MemoryBuffer {
	return &MemoryBuffer{
		buf: make(chan *entry.Entry, size),
		sem: semaphore.NewWeighted(size),
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
			m.inFlight.Store(id, e)
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
			m.inFlight.Store(id, e)
			inFlight[i] = id
		case <-timeout:
			return m.newFlushFunc(inFlight[:i]), i, nil
		}
	}

	return m.newFlushFunc(inFlight[:i]), i, nil
}

func (m *MemoryBuffer) newFlushFunc(ids []int64) func() {
	return func() {
		for _, id := range ids {
			m.inFlight.Delete(id)
		}
		m.sem.Release(int64(len(ids)))
	}
}
