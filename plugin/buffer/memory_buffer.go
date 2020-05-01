package buffer

import (
	"context"
	"fmt"

	"google.golang.org/api/support/bundler"
)

type MemoryBuffer struct {
	*bundler.Bundler
	cancel context.CancelFunc
}

func NewMemoryBuffer(entryType interface{}, handler func(context.Context, interface{}) error) *MemoryBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	handleFunc := func(entries interface{}) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := handler(ctx, entries)
			if err != nil {
				continue
			}

			break
		}
	}

	bd := bundler.NewBundler(entryType, handleFunc)
	bd.HandlerLimit = 16
	bd.BundleCountThreshold = 5000
	bd.BufferedByteLimit = 1024 * 1024 * 128
	return &MemoryBuffer{
		Bundler: bd,
		cancel:  cancel,
	}
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
