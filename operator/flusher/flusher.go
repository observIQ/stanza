package flusher

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// Config holds the configuration to build a new flusher
type Config struct {
	// MaxConcurrent is the maximum number of goroutines flushing entries concurrently.
	// Defaults to 16.
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`

	// MaxWait is the maximum amount of time to wait for a full slice of entries
	// before flushing the entries. Defaults to 1s.
	MaxWait helper.Duration `json:"max_wait" yaml:"max_wait"`

	// MaxChunkEntries is the maximum number of entries to flush at a time.
	// Defaults to 1000.
	MaxChunkEntries int `json:"max_chunk_entries" yaml:"max_chunk_entries"`

	// TODO configurable retry
}

// NewConfig creates a new default flusher config
func NewConfig() Config {
	return Config{
		MaxConcurrent: 16,
		MaxWait: helper.Duration{
			Duration: time.Second,
		},
		MaxChunkEntries: 1000,
	}
}

// Build uses a Config to build a new Flusher
func (c *Config) Build(buf buffer.Buffer, f FlushFunc, logger *zap.SugaredLogger) *Flusher {
	maxConcurrent := c.MaxConcurrent
	if maxConcurrent == 0 {
		maxConcurrent = 4
	}

  maxWait := c.MaxWait.Raw()
  if maxWait == time.Duration(0) {
    maxWait = time.Second
	}

	maxChunkEntries := c.MaxChunkEntries
	if maxChunkEntries == 0 {
		maxChunkEntries = 1000
	}

	return &Flusher{
		buffer:        buf,
		sem:           semaphore.NewWeighted(int64(maxConcurrent)),
		flush:         f,
		SugaredLogger: logger,
		waitTime:      maxWait,
		entrySlicePool: sync.Pool{
			New: func() interface{} {
				slice := make([]*entry.Entry, maxChunkEntries)
				return &slice
			},
		},
	}
}

// Flusher is used to flush entries from a buffer concurrently. It handles max concurrenty,
// retry behavior, and cancellation.
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

// FlushFunc is a function that the flusher uses to flush a slice of entries
type FlushFunc func(context.Context, []*entry.Entry) error

// Start begins flushing
func (f *Flusher) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.read(ctx)
	}()
}

// Stop cancels all the in-progress flushers and waits until they have returned
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

		// Fill a slice of entries
		entries := f.getEntrySlice()
		readCtx, cancel := context.WithTimeout(ctx, f.waitTime)
		markFlushed, n, err := f.buffer.ReadWait(readCtx, entries)
		cancel()
		if err != nil {
			f.Errorw("Failed to read entries from buffer", zap.Error(err))
		}

		// If we've timed out, but have no entries, don't bother flushing them
		if n == 0 {
			continue
		}

		// Wait until we have free flusher goroutines
		err = f.sem.Acquire(ctx, 1)
		if err != nil {
			// Context cancelled
			return
		}

		// Start a new flusher goroutine
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer f.sem.Release(1)
			defer f.putEntrySlice(entries)

			if err := f.flushWithRetry(ctx, entries[:n]); err == nil {
				if err := markFlushed(); err != nil {
					f.Errorw("Failed while marking entries flushed", zap.Error(err))
				}
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
	b := newExponentialBackoff()
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
		f.Warnw("Failed flushing chunk. Waiting before retry", "error", err, "wait_time", waitTime)

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

// newExponentialBackoff returns a default ExponentialBackOff
func newExponentialBackoff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     backoff.DefaultInitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         10 * time.Minute,
		MaxElapsedTime:      time.Duration(0),
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}
