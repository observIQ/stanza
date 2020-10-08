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
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)
	op := ops[0]
	require.IsType(t, &DropOutput{}, op)
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
	ops, err := cfg.Build(ctx)
	require.NoError(t, err)
	op := ops[0]

	entry := entry.New()
	result := op.Process(context.Background(), entry)
	require.Nil(t, result)
}
