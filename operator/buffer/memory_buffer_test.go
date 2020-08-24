package buffer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
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
		cfg := NewConfig()
		cfg.DelayThreshold = operator.Duration{Duration: 10 * time.Millisecond}
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
		cfg := NewConfig()
		cfg.DelayThreshold = operator.Duration{Duration: 10 * time.Millisecond}
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
		cfg := NewConfig()
		cfg.DelayThreshold = operator.Duration{Duration: 10 * time.Millisecond}
		cfg.Retry.MaxElapsedTime = operator.Duration{Duration: time.Nanosecond}
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

func TestMemoryBufferFlush(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		cfg := NewConfig()
		cfg.DelayThreshold = operator.Duration{Duration: 10 * time.Hour}
		buffer, err := cfg.Build()
		require.NoError(t, err)
		handler := newMockHandler(t)
		buffer.SetHandler(handler)

		err = buffer.Process(context.Background(), entry.New())
		require.NoError(t, err)

		// We shouldn't have any logs to handle for at
		// least 10 hours
		select {
		case <-handler.received:
			require.FailNow(t, "Received entry unexpectedly early")
		case <-time.After(50 * time.Millisecond):
		}

		flushed := make(chan struct{})
		go func() {
			defer close(flushed)
			err := buffer.Flush(context.Background())
			require.NoError(t, err)
		}()

		// After flushed is called, we should receive a log
		<-handler.received
		handler.fail <- false
		<-handler.success
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		cfg := NewConfig()
		cfg.DelayThreshold = operator.Duration{Duration: 10 * time.Hour}
		buffer, err := cfg.Build()
		require.NoError(t, err)
		handler := newMockHandler(t)
		buffer.SetHandler(handler)

		err = buffer.Process(context.Background(), entry.New())
		require.NoError(t, err)

		// We shouldn't have any logs to handle for at
		// least 10 hours
		select {
		case <-handler.received:
			require.FailNow(t, "Received entry unexpectedly early")
		case <-time.After(50 * time.Millisecond):
		}

		// Start the flush
		ctx, cancel := context.WithCancel(context.Background())
		flushed := make(chan struct{})
		go func() {
			defer close(flushed)
			err := buffer.Flush(ctx)
			require.Error(t, err)
		}()

		// Cancel the context and wait for flush to finish
		cancel()
		select {
		case <-flushed:
		case <-time.After(100 * time.Millisecond):
			require.FailNow(t, "Failed to flush in reasonable amount of time")
		}

		// After flushed is called, we should receive a log still, since we
		// timed out and ignored cleanup
		<-handler.received
		handler.fail <- false
		<-handler.success
	})

}
