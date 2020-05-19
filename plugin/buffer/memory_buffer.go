package buffer

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"google.golang.org/api/support/bundler"
)

type MemoryBuffer struct {
	*bundler.Bundler
	config *BufferConfig
	cancel context.CancelFunc
}

func NewMemoryBuffer(config *BufferConfig) *MemoryBuffer {
	return &MemoryBuffer{config: config}
}

type BundleHandler interface {
	ProcessMulti(context.Context, []*entry.Entry) error
	Logger() *zap.SugaredLogger
}

func (m *MemoryBuffer) SetHandler(handler BundleHandler) {
	ctx, cancel := context.WithCancel(context.Background())
	currentBundleID := int64(0)
	handleFunc := func(entries interface{}) {
		bundleID := atomic.AddInt64(&currentBundleID, 1)
		b := backoff.NewExponentialBackOff()
		for {
			err := handler.ProcessMulti(ctx, entries.([]*entry.Entry))
			if err != nil {
				duration := b.NextBackOff()
				if duration == backoff.Stop {
					handler.Logger().Errorw("Failed to flush bundle. Not retrying because we are beyond max backoff", zap.Error(err), "bundle_id", bundleID)
					break
				} else {
					handler.Logger().Warnw("Failed to flush bundle", zap.Error(err), "backoff_time", duration.String(), "bundle_id", bundleID)
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

func (b *MemoryBuffer) Flush(ctx context.Context) error {
	finished := make(chan struct{})
	go func() {
		b.Bundler.Flush()
		close(finished)
	}()

	select {
	case <-finished:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before flush finished")
	}
}

func (b *MemoryBuffer) Process(ctx context.Context, entry *entry.Entry) error {
	if b.Bundler == nil {
		panic("must call SetHandler before any calls to Process")
	}

	return b.AddWait(ctx, entry, 100) // TODO calculate size accurately?
}
