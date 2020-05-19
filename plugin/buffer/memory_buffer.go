package buffer

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/cenkalti/backoff/v4"
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

func (m *MemoryBuffer) SetHandler(handler func(context.Context, []*entry.Entry) error) {
	ctx, cancel := context.WithCancel(context.Background())
	handleFunc := func(entries interface{}) {
		b := backoff.NewExponentialBackOff()
		for {
			err := handler(ctx, entries.([]*entry.Entry))
			if err != nil {
				duration := b.NextBackOff()
				if duration == backoff.Stop {
					break
				} else {
					select {
					case <-ctx.Done():
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

	return b.AddWait(ctx, entry, 0) // TODO calculate size?
}
