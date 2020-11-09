package ratelimit

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	cfg := NewRateLimitConfig("my_rate_limit")
	cfg.OutputIDs = []string{"fake"}
	cfg.Burst = 1
	cfg.Rate = 1000

	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]

	fake := testutil.NewFakeOutput(t)

	err = op.SetOutputs([]operator.Operator{fake})
	require.NoError(t, err)

	err = op.Start()
	defer op.Stop()
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, ok := <-fake.Received
			if !ok {
				return
			}
		}
	}()

	// Warm up
	for i := 0; i < 100; i++ {
		err := op.Process(context.Background(), entry.New())
		require.NoError(t, err)
	}

	// Measure
	start := time.Now()
	for i := 0; i < 1000; i++ {
		err := op.Process(context.Background(), entry.New())
		require.NoError(t, err)
	}
	elapsed := time.Since(start)

	close(fake.Received)
	wg.Wait()

	if runtime.GOOS == "darwin" {
		t.Log("Using a wider acceptable range on darwin because of slow CI servers")
		require.InEpsilon(t, elapsed.Nanoseconds(), time.Second.Nanoseconds(), 0.6)
	} else {
		require.InEpsilon(t, elapsed.Nanoseconds(), time.Second.Nanoseconds(), 0.4)
	}
}
