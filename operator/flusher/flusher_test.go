package flusher

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestFlusher(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	buf, err := buffer.NewConfig().Build(buildContext, "testID")
	require.NoError(t, err)
	defer buf.Close()

	outChan := make(chan struct{})
	flushFunc := func(ctx context.Context, entries []*entry.Entry) error {
		for i := 0; i < len(entries); i++ {
			select {
			case <-ctx.Done():
				return nil
			case outChan <- struct{}{}:
			}
		}
		return nil
	}

	flusherCfg := NewConfig()
	flusherCfg.MaxWait = helper.Duration{
		Duration: 10 * time.Millisecond,
	}
	flusher := flusherCfg.Build(buf, flushFunc, nil)

	for i := 0; i < 100; i++ {
		err := buf.Add(context.Background(), entry.New())
		require.NoError(t, err)
	}

	flusher.Start()
	defer flusher.Stop()
	for i := 0; i < 100; i++ {
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "timed out")
		case <-outChan:
		}
	}
}
