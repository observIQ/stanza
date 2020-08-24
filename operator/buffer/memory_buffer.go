package buffer

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/observiq/stanza/entry"
	"go.uber.org/zap"
	"google.golang.org/api/support/bundler"
)

// MemoryBuffer is a buffer that holds entries in memory
type MemoryBuffer struct {
	*bundler.Bundler
	config *Config
	cancel context.CancelFunc
}

// NewMemoryBuffer will return a new memory buffer with the supplied configuration
func NewMemoryBuffer(config *Config) *MemoryBuffer {
	return &MemoryBuffer{config: config}
}

// BundleHandler is an interface that process multiple entries
type BundleHandler interface {
	ProcessMulti(context.Context, []*entry.Entry) error
	Logger() *zap.SugaredLogger
}

// SetHandler will set the handler of the memory buffer
func (m *MemoryBuffer) SetHandler(handler BundleHandler) {
	ctx, cancel := context.WithCancel(context.Background())
	currentBundleID := int64(0)
	handleFunc := func(entries interface{}) {
		bundleID := atomic.AddInt64(&currentBundleID, 1)
		b := m.NewExponentialBackOff()
		for {
			err := handler.ProcessMulti(ctx, entries.([]*entry.Entry))
			if err != nil {
				duration := b.NextBackOff()
				if duration == backoff.Stop {
					handler.Logger().Errorw("Failed to flush bundle. Not retrying because we are beyond max backoff", zap.Any("error", err), "bundle_id", bundleID)
					break
				} else {
					handler.Logger().Warnw("Failed to flush bundle", zap.Any("error", err), "backoff_time", duration.String(), "bundle_id", bundleID)
					select {
					case <-ctx.Done():
						handler.Logger().Debugw("Flush retry cancelled by context", "bundle_id", bundleID)
						return
					case <-time.After(duration):
						continue
					}
				}
			}

			break
		}
	}

	bd := bundler.NewBundler(&entry.Entry{}, handleFunc)
	bd.DelayThreshold = m.config.DelayThreshold.Raw()
	bd.BundleCountThreshold = m.config.BundleCountThreshold
	bd.BundleByteThreshold = m.config.BundleByteThreshold
	bd.BundleByteLimit = m.config.BundleByteLimit
	bd.BufferedByteLimit = m.config.BufferedByteLimit
	bd.HandlerLimit = m.config.HandlerLimit

	m.Bundler = bd
	m.cancel = cancel
}

// Flush will flush the memory buffer
func (m *MemoryBuffer) Flush(ctx context.Context) error {
	finished := make(chan struct{})
	go func() {
		m.Bundler.Flush()
		close(finished)
	}()

	select {
	case <-finished:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before flush finished")
	}
}

// Process will add an entry to the current buffer
func (m *MemoryBuffer) Process(ctx context.Context, entry *entry.Entry) error {
	if m.Bundler == nil {
		panic("must call SetHandler before any calls to Process")
	}

	return m.AddWait(ctx, entry, 100)
}

// NewExponentialBackOff will return a new exponential backoff for the memory buffer to use
func (m *MemoryBuffer) NewExponentialBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     m.config.Retry.InitialInterval.Raw(),
		RandomizationFactor: m.config.Retry.RandomizationFactor,
		Multiplier:          m.config.Retry.Multiplier,
		MaxInterval:         m.config.Retry.MaxInterval.Raw(),
		MaxElapsedTime:      m.config.Retry.MaxElapsedTime.Raw(),
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}
