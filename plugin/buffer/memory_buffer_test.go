package buffer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type mockHandler struct {
	fail     chan bool
	received chan []*entry.Entry
	success  chan []*entry.Entry
	logger   *zap.SugaredLogger
}

func (h *mockHandler) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {
	h.received <- entries
	fail := <-h.fail
	if fail {
		return fmt.Errorf("test failure")
	}

	h.success <- entries
	return nil
}

func (h *mockHandler) Logger() *zap.SugaredLogger {
	return h.logger
}

func newMockHandler(t *testing.T) *mockHandler {
	return &mockHandler{
		fail:     make(chan bool),
		received: make(chan []*entry.Entry),
		success:  make(chan []*entry.Entry),
		logger:   zaptest.NewLogger(t).Sugar(),
	}
}

func TestMemoryBufferRetry(t *testing.T) {
	t.Run("FailOnce", func(t *testing.T) {
		cfg := &BufferConfig{
			DelayThreshold: plugin.Duration{10 * time.Millisecond},
		}
		buffer, err := cfg.Build()
		require.NoError(t, err)
		handler := newMockHandler(t)
		buffer.SetHandler(handler)

		err = buffer.Process(context.Background(), entry.New())
		require.NoError(t, err)

		// Tell it to fail as soon as we receive logs
		<-handler.received
		handler.fail <- true

		// The next time receive, don't fail, and ensure that we get a success
		<-handler.received
		handler.fail <- false
		<-handler.success
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		cfg := &BufferConfig{
			DelayThreshold: plugin.Duration{10 * time.Millisecond},
		}
		buffer, err := cfg.Build()
		require.NoError(t, err)
		handler := newMockHandler(t)
		buffer.SetHandler(handler)

		ctx, cancel := context.WithCancel(context.Background())
		err = buffer.Process(ctx, entry.New())
		require.NoError(t, err)

		// Fail once, but cancel the context so we don't retry
		<-handler.received
		cancel()
		handler.fail <- true

		// We shouldn't get any more receives or successes
		select {
		case <-handler.received:
			require.FailNow(t, "Received unexpected entries")
		case <-handler.success:
			require.FailNow(t, "Received unexpected success")
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("ExceededLimit", func(t *testing.T) {
		cfg := &BufferConfig{
			DelayThreshold: plugin.Duration{10 * time.Millisecond},
			Retry: RetryConfig{
				MaxElapsedTime: plugin.Duration{time.Nanosecond},
			},
		}
		buffer, err := cfg.Build()
		require.NoError(t, err)
		handler := newMockHandler(t)
		buffer.SetHandler(handler)

		err = buffer.Process(context.Background(), entry.New())
		require.NoError(t, err)

		// Fail once, which should exceed our
		// unreasonably low max elapsed time
		<-handler.received
		handler.fail <- true

		// We shouldn't get any more receives or successes
		select {
		case <-handler.received:
			require.FailNow(t, "Received unexpected entries")
		case <-handler.success:
			require.FailNow(t, "Received unexpected success")
		case <-time.After(200 * time.Millisecond):
		}
	})
}
