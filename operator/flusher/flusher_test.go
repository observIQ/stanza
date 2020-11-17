package flusher

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFlusher(t *testing.T) {
	outChan := make(chan struct{}, 100)
	flusherCfg := NewConfig()
	flusher := flusherCfg.Build(zaptest.NewLogger(t).Sugar())

	failed := errors.New("test failure")
	for i := 0; i < 100; i++ {
		flusher.Do(func(_ context.Context) error {
			// Fail randomly but still expect the entries to come through
			if rand.Int()%5 == 0 {
				return failed
			}
			outChan <- struct{}{}
			return nil
		})
	}

	for i := 0; i < 100; i++ {
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "timed out")
		case <-outChan:
		}
	}
}
