package drop

import (
	"context"
	"fmt"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func NewTestConfig(t *testing.T) (*operator.Config, error) {
	json := `{
		"type": "drop_output",
		"id": "test_id"
	}`
	config := &operator.Config{}
	err := config.UnmarshalJSON([]byte(json))
	return config, err
}

func NewTestOutput(t *testing.T) (*DropOutput, error) {
	config, err := NewTestConfig(t)
	if err != nil {
		return nil, err
	}

	ctx := testutil.NewBuildContext(t)
	op, err := config.Build(ctx)
	if err != nil {
		return nil, err
	}

	output, ok := op.(*DropOutput)
	if !ok {
		return nil, fmt.Errorf("operator is not a drop output")
	}

	return output, nil
}

func TestBuildValid(t *testing.T) {
	cfg, err := NewTestConfig(t)
	require.NoError(t, err)

	ctx := testutil.NewBuildContext(t)
	output, err := cfg.Build(ctx)
	require.NoError(t, err)
	require.IsType(t, &DropOutput{}, output)
}

func TestBuildIvalid(t *testing.T) {
	cfg, err := NewTestConfig(t)
	require.NoError(t, err)

	ctx := testutil.NewBuildContext(t)
	ctx.Logger = nil
	_, err = cfg.Build(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build context is missing a logger")
}

func TestProcess(t *testing.T) {
	output, err := NewTestOutput(t)
	require.NoError(t, err)
	
	entry := entry.New()
	ctx := context.Background()
	result := output.Process(ctx, entry)
	require.Nil(t, result)
}
