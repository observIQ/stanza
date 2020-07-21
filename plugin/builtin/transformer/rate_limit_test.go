package transformer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	cfg := RateLimitConfig{
		TransformerConfig: helper.TransformerConfig{
			WriterConfig: helper.WriterConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "my_rate_limit",
					PluginType: "rate_limit",
				},
				OutputIDs: []string{"output1"},
			},
		},

		Rate:  10,
		Burst: 1,
	}

	rateLimit, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	receivedLog := make(chan struct{}, 100)
	mockOutput := testutil.NewMockPlugin("output1")
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedLog <- struct{}{}
	})

	err = rateLimit.SetOutputs([]plugin.Plugin{mockOutput})
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
