package flusher

import (
	"context"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	// maxConcurrency is the default maximum amount of concurrent flush operations
	maxConcurrency = 16

	// maxRetryInterval is the default maximum retry interval duration
	maxRetryInterval = time.Minute

	// maxElapsedTime is the default maximum duration to attempt retries
	maxElapsedTime = time.Hour
)

// Config holds the configuration to build a new flusher
type Config struct {
	// MaxConcurrent is the maximum number of goroutines flushing entries concurrently.
	// Defaults to 16.
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`

	// Retry Config

	// MaxRetryTime maximum duration to continue retrying for
	// Defaults to 1 Hour
	MaxRetryTime time.Duration `json:"max_retry_time" yaml:"max_retry_time"`

	// MaxRetryInterval the maximum retry interval duration.
	// Defaults to 1 Minute
	MaxRetryInterval time.Duration `json:"max_retry_interval" yaml:"max_retry_interval"`
}

// NewConfig creates a new default flusher config
func NewConfig() Config {
	return Config{
		MaxConcurrent:    maxConcurrency,
		MaxRetryTime:     maxElapsedTime,
		MaxRetryInterval: maxRetryInterval,
	}
}

// Build uses a Config to build a new Flusher
func (c Config) Build(logger *zap.SugaredLogger) *Flusher {
	concurrency := c.MaxConcurrent
	if concurrency == 0 || concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}

	elapsedMax := c.MaxRetryTime
	if elapsedMax == 0 || elapsedMax > maxElapsedTime {
		elapsedMax = maxElapsedTime
	}

	retryIntervalMax := c.MaxRetryInterval
	if retryIntervalMax == 0 || retryIntervalMax > maxRetryInterval {
		retryIntervalMax = maxRetryInterval
	}

	return &Flusher{
		sem:              semaphore.NewWeighted(int64(concurrency)),
		doneChan:         make(chan struct{}),
		SugaredLogger:    logger,
		maxElapsedTime:   elapsedMax,
		maxRetryInterval: retryIntervalMax,
	}
}

// Flusher is used to flush entries from a buffer concurrently. It handles max concurrency,
// retry behavior, and cancellation.
type Flusher struct {
	doneChan chan struct{}
	sem      *semaphore.Weighted
	wg       sync.WaitGroup
	*zap.SugaredLogger

	// Retry maximums
	maxElapsedTime   time.Duration
	maxRetryInterval time.Duration
}

// FlushFunc is any function that flushes
type FlushFunc func(context.Context) error

// Do executes the flusher function in a goroutine
func (f *Flusher) Do(ctx context.Context, flush FlushFunc) {
	// Wait until we have free flusher goroutines
	if err := f.sem.Acquire(ctx, 1); err != nil {
		// Context cancelled
		return
	}

	// Start a new flusher goroutine
	f.wg.Add(1)
	go func() {
		defer f.sem.Release(1)
		defer f.wg.Done()
		f.flushWithRetry(ctx, flush)
	}()
}

// Stop cancels all the in-progress flushers and waits until they have returned
func (f *Flusher) Stop() {
	close(f.doneChan)
	f.wg.Wait()
}

// flushWithRetry will run flush with a backoff until one of the following occurs:
//   - flush exits without an error
//   - Backoff hits maximumRetryTime
//   - passed in context cancels
//   - flusher is stopped
func (f *Flusher) flushWithRetry(ctx context.Context, flush FlushFunc) {
	b := f.newExponentialBackoff()

	// Initialize wait time with no duration so first wait immediately executes
	waitTime := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			f.Debugw("flushWithRetry context has canceled", zap.Error(ctx.Err()))
			return
		case <-f.doneChan:
			f.Debug("flusher has stopped, ending retry")
			return
		case <-time.After(waitTime):
			// Attempt to flush, return if successful
			err := flush(ctx)
			if err == nil {
				return
			}

			waitTime = b.NextBackOff()
			// Maximum wait time reached
			if waitTime == b.Stop {
				f.Debug("flusherWithRetry has reached maximum retry")
				return
			}

			f.Warnw("Failed flushing. Waiting before retry", zap.Error(err), "wait_time", waitTime)
		}
	}
}

// newExponentialBackoff returns a default ExponentialBackOff
func (f *Flusher) newExponentialBackoff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     50 * time.Millisecond,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         f.maxRetryInterval,
		MaxElapsedTime:      f.maxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}
