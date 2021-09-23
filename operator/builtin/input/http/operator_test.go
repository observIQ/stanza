package httpevents

import (
	"testing"

	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = "localhost:8080"
	op, err := cfg.build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.NoError(t, op.Start(), "failed to start operator")
	require.NoError(t, op.Stop(), "failed to stop operator")

	// stopping again should not panic
	p := func() {
		op.Stop()
	}
	require.NotPanics(t, p)
}
