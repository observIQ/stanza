package noop

import (
	"context"
	"testing"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func NewTestConfig(t *testing.T) (*operator.Config, error) {
	json := `{
		"type": "noop",
		"id": "test_id",
		"output": "test_output"
	}`
	config := &operator.Config{}
	err := config.UnmarshalJSON([]byte(json))
	return config, err
}

func TestBuildValid(t *testing.T) {
	cfg := NewNoopOperatorConfig("test")
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]
	require.IsType(t, &NoopOperator{}, op)
}

func TestBuildIvalid(t *testing.T) {
	cfg := NewNoopOperatorConfig("test")
	ctx := testutil.NewBuildContext(t)
	ctx.Logger = nil
	_, err := cfg.Build(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build context is missing a logger")
}

func TestProcess(t *testing.T) {
	cfg := NewNoopOperatorConfig("test")
	cfg.OutputIDs = []string{"fake"}
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]

	fake := testutil.NewFakeOutput(t)
	op.SetOutputs([]operator.Operator{fake})

	entry := entry.New()
	entry.AddLabel("label", "value")
	entry.AddResourceKey("resource", "value")

	expected := entry.Copy()
	err = op.Process(context.Background(), entry)
	require.NoError(t, err)

	fake.ExpectEntry(t, expected)
}
