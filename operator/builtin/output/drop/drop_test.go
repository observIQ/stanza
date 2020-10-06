package drop

import (
	"context"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuildValid(t *testing.T) {
	cfg := NewDropOutputConfig("test")
	ctx := testutil.NewBuildContext(t)
	output, err := cfg.Build(ctx)
	require.NoError(t, err)
	require.IsType(t, &DropOutput{}, output)
}

func TestBuildIvalid(t *testing.T) {
	cfg := NewDropOutputConfig("test")
	ctx := testutil.NewBuildContext(t)
	ctx.Logger = nil
	_, err := cfg.Build(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build context is missing a logger")
}

func TestProcess(t *testing.T) {
	cfg := NewDropOutputConfig("test")
	ctx := testutil.NewBuildContext(t)
	output, err := cfg.Build(ctx)
	require.NoError(t, err)

	entry := entry.New()
	result := output.Process(context.Background(), entry)
	require.Nil(t, result)
}
