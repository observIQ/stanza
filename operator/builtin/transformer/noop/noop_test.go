package noop

import (
	"context"
	"fmt"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
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

func NewTestOperator(t *testing.T) (*NoopOperator, error) {
	config, err := NewTestConfig(t)
	if err != nil {
		return nil, err
	}

	ctx := testutil.NewBuildContext(t)
	op, err := config.Build(ctx)
	if err != nil {
		return nil, err
	}

	noop, ok := op.(*NoopOperator)
	if !ok {
		return nil, fmt.Errorf("operator is not a json parser")
	}

	return noop, nil
}

func TestBuildValid(t *testing.T) {
	cfg, err := NewTestConfig(t)
	require.NoError(t, err)

	ctx := testutil.NewBuildContext(t)
	output, err := cfg.Build(ctx)
	require.NoError(t, err)
	require.IsType(t, &NoopOperator{}, output)
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
	noop, err := NewTestOperator(t)
	require.NoError(t, err)

	var processedEntry interface{}
	mockOutput := testutil.NewMockOperator("test_output")
	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) { processedEntry = args[1] }).Return(nil)
	noop.OutputOperators = []operator.Operator{mockOutput}

	entry := entry.New()
	entry.AddLabel("label", "value")
	entry.AddResourceKey("resource", "value")

	expected := entry.Copy()
	ctx := context.Background()
	result := noop.Process(ctx, entry)
	require.Nil(t, result)
	require.Equal(t, expected, processedEntry)
}
