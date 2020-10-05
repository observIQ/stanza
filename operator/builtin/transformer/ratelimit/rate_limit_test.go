package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	cfg := NewRateLimitConfig("my_rate_limit")
	cfg.OutputIDs = []string{"fake"}
	cfg.Burst = 1
	cfg.Rate = 1000

	rateLimit, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	fake := testutil.NewFakeOutput(t)

	err = rateLimit.SetOutputs([]operator.Operator{fake})
	require.NoError(t, err)

	err = rateLimit.Start()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
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

	// This allows for the operator to reach steady operation
	timeout := time.After(100 * time.Millisecond)
WARMUP:
	for {
		select {
		case <-fake.Received:
			// Just consume the channel to keep it empty
		case <-timeout:
			break WARMUP
		}
	}

	i := 0
LOOP:
	for {
		select {
		case <-fake.Received:
			i++
		case <-timeout:
			break LOOP
		}
	}

	cancel()
	wg.Wait()

	require.InDelta(t, 100, i, 20)
}
