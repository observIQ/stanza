package helper

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

func TestTransformerConfigMissingBase(t *testing.T) {
	cfg := NewTransformerConfig("test", "")
	cfg.OutputIDs = []string{"test-output"}
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestTransformerConfigMissingOutput(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}

func TestTransformerConfigValid(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	cfg.OutputIDs = []string{"test-output"}
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}

func TestTransformerOnErrorDefault(t *testing.T) {
	cfg := NewTransformerConfig("test-id", "test-type")
	transformer, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.Equal(t, SendOnError, transformer.OnError)
}

func TestTransformerOnErrorInvalid(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	cfg.OnError = "invalid"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator config has an invalid `on_error` field.")
}

func TestTransformerOperatorCanProcess(t *testing.T) {
	cfg := NewTransformerConfig("test", "test")
	transformer, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.True(t, transformer.CanProcess())
}

func TestTransformerDropOnError(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerOperator{
		OnError: DropOnError,
		WriterOperator: WriterOperator{
			BasicOperator: BasicOperator{
				OperatorID:    "test-id",
				OperatorType:  "test-type",
				SugaredLogger: buildContext.Logger.SugaredLogger,
			},
			OutputOperators: []operator.Operator{output},
			OutputIDs:       []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) error {
		return fmt.Errorf("Failure")
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.Error(t, err)
	output.AssertNotCalled(t, "Process", mock.Anything, mock.Anything)
}

func TestTransformerSendOnError(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerOperator{
		OnError: SendOnError,
		WriterOperator: WriterOperator{
			BasicOperator: BasicOperator{
				OperatorID:    "test-id",
				OperatorType:  "test-type",
				SugaredLogger: buildContext.Logger.SugaredLogger,
			},
			OutputOperators: []operator.Operator{output},
			OutputIDs:       []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) error {
		return fmt.Errorf("Failure")
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.NoError(t, err)
	output.AssertCalled(t, "Process", mock.Anything, mock.Anything)
}

func TestTransformerProcessWithValid(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	transformer := TransformerOperator{
		OnError: SendOnError,
		WriterOperator: WriterOperator{
			BasicOperator: BasicOperator{
				OperatorID:    "test-id",
				OperatorType:  "test-type",
				SugaredLogger: buildContext.Logger.SugaredLogger,
			},
			OutputOperators: []operator.Operator{output},
			OutputIDs:       []string{"test-output"},
		},
	}
	ctx := context.Background()
	testEntry := entry.New()
	transform := func(e *entry.Entry) error {
		return nil
	}

	err := transformer.ProcessWith(ctx, testEntry, transform)
	require.NoError(t, err)
	output.AssertCalled(t, "Process", mock.Anything, mock.Anything)
}
