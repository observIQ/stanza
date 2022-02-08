package buffer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreedyCountingSemaphoreIncrement(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Test increment no readers",
			testFunc: func(t *testing.T) {
				t.Parallel()
				rs := NewGreedyCountingSemaphore(0)
				rs.Increment()
				require.Equal(t, int64(1), rs.val)
			},
		},
		{
			desc: "Test increment waiting readers",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(0)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, time.Hour, 1)
					assert.Equal(t, int64(1), amnt)
					close(doneChan)
				}()

				<-time.After(50 * time.Millisecond)

				rs.Increment()
				require.Equal(t, int64(0), rs.val)
				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out waiting for acquire")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestGreedyCountingSemaphoreAcquire(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Acquire blocks when 0",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(0)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, time.Hour, 1)
					assert.Equal(t, int64(0), amnt)
					close(doneChan)
				}()

				<-time.After(50 * time.Millisecond)

				select {
				case <-doneChan:
					require.Fail(t, "Acquired semaphore despite not incrementing")
				case <-time.After(timeout):
				}
			},
		},
		{
			desc: "Acquire works when semaphore val is 1",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(1)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, time.Hour, 1)
					assert.Equal(t, int64(1), amnt)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out acquiring semaphore")
				}
			},
		},
		{
			desc: "Acquire returns when context cancelled",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(0)

				ctx, cancel := context.WithCancel(context.Background())

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, time.Hour, 1)
					assert.Equal(t, int64(0), amnt)
					close(doneChan)
				}()

				<-time.After(50 * time.Millisecond)

				cancel()

				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out acquiring semaphore")
				}
			},
		},
		{
			desc: "Acquire returns when timeout",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(0)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, 50*time.Millisecond, 1)
					assert.Equal(t, int64(0), amnt)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out acquiring semaphore")
				}
			},
		},
		{
			desc: "Acquire is greedy on context cancel",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(25)

				ctx, cancel := context.WithCancel(context.Background())

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, time.Hour, 100)
					assert.Equal(t, int64(25), amnt)
					close(doneChan)
				}()

				<-time.After(50 * time.Millisecond)

				cancel()

				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out acquiring semaphore")
				}
			},
		},
		{
			desc: "Acquire is greedy on timeout",
			testFunc: func(t *testing.T) {
				t.Parallel()
				timeout := 250 * time.Millisecond
				rs := NewGreedyCountingSemaphore(25)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				doneChan := make(chan struct{})
				go func() {
					amnt := rs.AcquireAtMost(ctx, 50*time.Millisecond, 100)
					assert.Equal(t, int64(25), amnt)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-time.After(timeout):
					require.Fail(t, "timed out acquiring semaphore")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
