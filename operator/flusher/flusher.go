package flusher

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type Config struct {
	MaxConcurrent   int               `json:"max_concurrent" yaml:"max_concurrent"`
	MaxWait         operator.Duration `json:"max_wait" yaml:"max_wait"`
	MaxChunkEntries int               `json:"max_chunk_entries" yaml:"max_chunk_entries"`
}

func NewConfig() Config {
	return Config{
		MaxConcurrent: 16,
		MaxWait: operator.Duration{
			Duration: time.Second,
		},
		MaxChunkEntries: 1000,
	}
}

func (c *Config) Build(buf buffer.Buffer, f FlushFunc, logger *zap.SugaredLogger) *Flusher {
	return &Flusher{
		buffer:        buf,
		sem:           semaphore.NewWeighted(int64(c.MaxConcurrent)),
		flush:         f,
		SugaredLogger: logger,
		waitTime:      c.MaxWait.Duration,
		entrySlicePool: sync.Pool{
			New: func() interface{} {
				slice := make([]*entry.Entry, c.MaxChunkEntries)
				return &slice
			},
		},
	}
}

type Flusher struct {
	buffer         buffer.Buffer
	sem            *semaphore.Weighted
	wg             sync.WaitGroup
	cancel         context.CancelFunc
	chunkIDCounter uint64
	flush          FlushFunc
	waitTime       time.Duration
	entrySlicePool sync.Pool
	*zap.SugaredLogger
}

type FlushFunc func(context.Context, []*entry.Entry) error

func (f *Flusher) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.read(ctx)
	}()
}

func (f *Flusher) Stop() {
	f.cancel()
	f.wg.Wait()
}

func (f *Flusher) read(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		entries := f.getEntrySlice()
		readCtx, cancel := context.WithTimeout(ctx, f.waitTime)
		markFlushed, n, err := f.buffer.ReadWait(readCtx, entries)
		cancel()
		if err != nil {
			f.Errorw("Failed to read entries from buffer", zap.Error(err))
		}

		if n == 0 {
			continue
		}

		err = f.sem.Acquire(ctx, 1)
		if err != nil {
			// Context cancelled
			return
		}

		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer f.sem.Release(1)
			defer f.putEntrySlice(entries)

			err := f.flushWithRetry(ctx, entries[:n])
			if err == nil {
				markFlushed()
			}
		}()
	}
}

// flushWithRetry will continue trying to call Flusher.flushFunc with the entries passed
// in until either flushFunc returns no error or the context is cancelled. It will only
// return an error in the case that the context was cancelled. If no error was returned,
// it is safe to mark the entries in the buffer as flushed.
func (f *Flusher) flushWithRetry(ctx context.Context, entries []*entry.Entry) error {
	chunkID := atomic.AddUint64(&f.chunkIDCounter, 1)
	b := backoff.NewExponentialBackOff()
	for {
		err := f.flush(ctx, entries)
		if err == nil {
			return nil
		}

		waitTime := b.NextBackOff()
		if waitTime == b.Stop {
			f.Errorw("Reached max backoff time during chunk flush retry. Dropping logs in chunk", "chunk_id", chunkID)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

func (f *Flusher) getEntrySlice() []*entry.Entry {
	return *(f.entrySlicePool.Get().(*[]*entry.Entry))
}

func (f *Flusher) putEntrySlice(slice []*entry.Entry) {
	f.entrySlicePool.Put(&slice)
}
