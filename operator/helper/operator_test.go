package helper

import (
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasicConfigID(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-id", config.ID())
}

func TestBasicConfigType(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-type", config.Type())
}

func TestBasicConfigBuildWithoutID(t *testing.T) {
	config := BasicConfig{
		OperatorType: "test-type",
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestBasicConfigBuildWithoutType(t *testing.T) {
	config := BasicConfig{
		OperatorID: "test-id",
	}
	context := operator.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestBasicConfigBuildMissingLogger(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	context := operator.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator build context is missing a logger.")
}

func TestBasicConfigBuildValid(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	context := testutil.NewBuildContext(t)
	operator, err := config.Build(context)
	require.NoError(t, err)
	require.Equal(t, "test-id", operator.OperatorID)
	require.Equal(t, "test-type", operator.OperatorType)
}

func TestBasicOperatorID(t *testing.T) {
	operator := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-id", operator.ID())
}

func TestBasicOperatorType(t *testing.T) {
	operator := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-type", operator.Type())
}

func TestBasicOperatorLogger(t *testing.T) {
	logger := &zap.SugaredLogger{}
	operator := BasicOperator{
		OperatorID:    "test-id",
		OperatorType:  "test-type",
		SugaredLogger: logger,
	}
	require.Equal(t, logger, operator.Logger())
}

func TestBasicOperatorStart(t *testing.T) {
	operator := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	err := operator.Start()
	require.NoError(t, err)
}

func TestBasicOperatorStop(t *testing.T) {
	operator := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	err := operator.Stop()
	require.NoError(t, err)
}
