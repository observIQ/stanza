package buffer

import (
	"context"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBufferBuild(t *testing.T) {
	cfg := NewMemoryBufferConfig()
	operatorID := "operator"

	buffer, err := cfg.Build(operatorID)
	require.NoError(t, err)
	require.IsType(t, &MemoryBuffer{}, buffer)

	memBuffer := buffer.(*MemoryBuffer)
	assert.Equal(t, operatorID, memBuffer.operatorID)
	assert.Equal(t, cfg.MaxChunkDelay.Raw(), memBuffer.maxChunkDelay)
	assert.Equal(t, cfg.MaxChunkSize, memBuffer.maxChunkSize)
	assert.False(t, memBuffer.closed)
}

func TestMemoryBufferAdd(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Can not add to closed buffer",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Close buffer
				_, err = buffer.Close()
				require.NoError(t, err)

				// Attempt to add to buffer
				err = buffer.Add(context.Background(), &entry.Entry{})
				assert.ErrorIs(t, err, ErrBufferedClosed)
			},
		},
		{
			desc: "Context canceled",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				// Max entries 0 for a non buffered channel
				cfg.MaxEntries = 0
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Create a context with a deadline
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				// Make a timer to protect against the test hanging
				timer := time.NewTimer(3 * time.Second)
				defer timer.Stop()

				// Channel to signal test is done
				doneChan := make(chan struct{})

				go func() {
					err := buffer.Add(ctx, &entry.Entry{})
					assert.ErrorIs(t, err, context.DeadlineExceeded)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-timer.C:
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "Successful Add",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				// Max entries 0 for a non buffered channel
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				err = buffer.Add(context.Background(), &entry.Entry{})
				require.NoError(t, err)

				memBuffer := buffer.(*MemoryBuffer)
				assert.Len(t, memBuffer.buf, 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestMemoryBufferRead(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Can not read from closed buffer",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Close buffer
				_, err = buffer.Close()
				require.NoError(t, err)

				// Attempt to add to buffer
				_, err = buffer.Read(context.Background())
				assert.ErrorIs(t, err, ErrBufferedClosed)
			},
		},
		{
			desc: "Context canceled",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				// Create a large duration to ensure we never hit it
				cfg.MaxChunkDelay = helper.NewDuration(20 * time.Minute)
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Create a context with a deadline
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				// Make a timer to protect against the test hanging
				timer := time.NewTimer(3 * time.Second)
				defer timer.Stop()

				// Channel to signal test is done
				doneChan := make(chan struct{})

				go func() {
					_, err := buffer.Read(ctx)
					assert.ErrorIs(t, err, context.DeadlineExceeded)
					close(doneChan)
				}()

				select {
				case <-doneChan:
				case <-timer.C:
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "Max Chunk Size returns when reached",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				// Create a large duration to ensure we never hit it
				cfg.MaxChunkDelay = helper.NewDuration(20 * time.Minute)
				// Ensure max chunk size is large enough we won't hit it
				cfg.MaxChunkSize = 2
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Add two entries to buffer
				err = buffer.Add(context.Background(), &entry.Entry{})
				require.NoError(t, err)

				err = buffer.Add(context.Background(), &entry.Entry{})
				require.NoError(t, err)

				// Make a timer to protect against the test hanging
				timer := time.NewTimer(3 * time.Second)
				defer timer.Stop()

				// Channel to signal test is done
				doneChan := make(chan struct{})

				var entries []*entry.Entry
				go func() {
					entries, err = buffer.Read(context.Background())
					assert.NoError(t, err)
					close(doneChan)
				}()

				select {
				case <-doneChan:
					assert.Len(t, entries, 2)
				case <-timer.C:
					assert.Fail(t, "test timed out")
				}
			},
		},
		{
			desc: "Max Chunk Delay returns when hit",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewMemoryBufferConfig()
				// Create a large duration to ensure we never hit it
				cfg.MaxChunkDelay = helper.NewDuration(1 * time.Second)
				// Ensure max chunk size is large enough we won't hit it
				cfg.MaxChunkSize = 20
				buffer, err := cfg.Build("operatorID")
				require.NoError(t, err)

				// Add single entry to buffer
				err = buffer.Add(context.Background(), &entry.Entry{})
				require.NoError(t, err)

				// Make a timer to protect against the test hanging
				timer := time.NewTimer(3 * time.Second)
				defer timer.Stop()

				// Channel to signal test is done
				doneChan := make(chan struct{})

				var entries []*entry.Entry
				go func() {
					entries, err = buffer.Read(context.Background())
					assert.NoError(t, err)
					close(doneChan)
				}()

				select {
				case <-doneChan:
					assert.Len(t, entries, 1)
				case <-timer.C:
					assert.Fail(t, "test timed out")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
