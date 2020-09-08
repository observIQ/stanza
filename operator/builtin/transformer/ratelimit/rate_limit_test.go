package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	cfg := NewRateLimitConfig("my_rate_limit")
	cfg.OutputIDs = []string{"output1"}
	cfg.Burst = 1
	cfg.Rate = 10

	rateLimit, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	receivedLog := make(chan struct{}, 100)
	mockOutput := testutil.NewMockOperator("output1")
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedLog <- struct{}{}
	})

	err = rateLimit.SetOutputs([]operator.Operator{mockOutput})
	require.NoError(t, err)

	err = rateLimit.Start()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := rateLimit.Process(ctx, entry.New())
				require.NoError(t, err)
			}
		}
	}()

	i := 0
	timeout := time.After(time.Second)
LOOP:
	for {
		select {
		case <-receivedLog:
			i++
		case <-timeout:
			break LOOP
		}
	}

	cancel()
	wg.Wait()

	require.InDelta(t, 10, i, 3)
}
