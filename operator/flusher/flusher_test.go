package flusher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfigBuild(t *testing.T) {
	noopLogger := zap.NewNop().Sugar()
	testCases := []struct {
		desc     string
		config   Config
		expected *Flusher
	}{
		{
			desc:   "Blank config",
			config: Config{},
			expected: &Flusher{
				SugaredLogger:    noopLogger,
				maxElapsedTime:   maxElapsedTime,
				maxRetryInterval: maxRetryInterval,
			},
		},
		{
			desc:   "NewConfig()",
			config: NewConfig(),
			expected: &Flusher{
				SugaredLogger:    noopLogger,
				maxElapsedTime:   maxElapsedTime,
				maxRetryInterval: maxRetryInterval,
			},
		},
		{
			desc: "Valid non-default values",
			config: Config{
				MaxConcurrent:    1,
				MaxRetryTime:     2 * time.Minute,
				MaxRetryInterval: 3 * time.Second,
			},
			expected: &Flusher{
				SugaredLogger:    noopLogger,
				maxElapsedTime:   2 * time.Minute,
				maxRetryInterval: 3 * time.Second,
			},
		},
		{
			desc: "To large non-default values",
			config: Config{
				MaxConcurrent:    maxConcurrency + 1,
				MaxRetryTime:     maxElapsedTime * 2,
				MaxRetryInterval: maxRetryInterval * 3,
			},
			expected: &Flusher{
				SugaredLogger:    noopLogger,
				maxElapsedTime:   maxElapsedTime,
				maxRetryInterval: maxRetryInterval,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			flusher := tc.config.Build(noopLogger)

			// Only check some of the fields as others don't equal during an equality check
			assert.Equal(t, tc.expected.maxElapsedTime, flusher.maxElapsedTime)
			assert.Equal(t, tc.expected.maxRetryInterval, flusher.maxRetryInterval)
			assert.Equal(t, noopLogger, flusher.SugaredLogger)
		})
	}
}

func TestFlusher_flushWithRetry(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Context Canceled",
			testFunc: func(t *testing.T) {
				flusher := NewConfig().Build(zap.NewNop().Sugar())

				doneChan := make(chan struct{})
				flush := func(ctx context.Context) error {
					<-doneChan
					return nil
				}

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				go func() {
					flusher.flushWithRetry(ctx, flush)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(5 * time.Second):
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "Flusher Closed",
			testFunc: func(t *testing.T) {
				flusher := NewConfig().Build(zap.NewNop().Sugar())

				doneChan := make(chan struct{})
				flush := func(ctx context.Context) error {
					<-doneChan
					return nil
				}

				// Call stop to close doneChan
				flusher.Stop()
				go func() {
					flusher.flushWithRetry(context.Background(), flush)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(5 * time.Second):
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "Hit max amount of retries",
			testFunc: func(t *testing.T) {
				config := NewConfig()
				config.MaxRetryTime = 1 * time.Second
				flusher := config.Build(zap.NewNop().Sugar())

				doneChan := make(chan struct{})
				flush := func(ctx context.Context) error {
					return errors.New("bad")
				}

				go func() {
					flusher.flushWithRetry(context.Background(), flush)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(5 * time.Second):
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "No retry needed",
			testFunc: func(t *testing.T) {
				flusher := NewConfig().Build(zap.NewNop().Sugar())

				doneChan := make(chan struct{})
				flush := func(ctx context.Context) error {
					return nil
				}

				go func() {
					flusher.flushWithRetry(context.Background(), flush)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(5 * time.Second):
					assert.Fail(t, "test timed out")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
