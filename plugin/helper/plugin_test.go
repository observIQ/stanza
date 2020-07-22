package helper

import (
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
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
	context := plugin.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestBasicConfigBuildMissingLogger(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	context := plugin.BuildContext{}
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin build context is missing a logger.")
}

func TestBasicConfigBuildValid(t *testing.T) {
	config := BasicConfig{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	context := testutil.NewBuildContext(t)
	plugin, err := config.Build(context)
	require.NoError(t, err)
	require.Equal(t, "test-id", plugin.OperatorID)
	require.Equal(t, "test-type", plugin.OperatorType)
}

func TestBasicOperatorID(t *testing.T) {
	plugin := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-id", plugin.ID())
}

func TestBasicOperatorType(t *testing.T) {
	plugin := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	require.Equal(t, "test-type", plugin.Type())
}

func TestBasicOperatorLogger(t *testing.T) {
	logger := &zap.SugaredLogger{}
	plugin := BasicOperator{
		OperatorID:    "test-id",
		OperatorType:  "test-type",
		SugaredLogger: logger,
	}
	require.Equal(t, logger, plugin.Logger())
}

func TestBasicOperatorStart(t *testing.T) {
	plugin := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	err := plugin.Start()
	require.NoError(t, err)
}

func TestBasicOperatorStop(t *testing.T) {
	plugin := BasicOperator{
		OperatorID:   "test-id",
		OperatorType: "test-type",
	}
	err := plugin.Stop()
	require.NoError(t, err)
}
