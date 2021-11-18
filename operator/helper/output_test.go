package helper

import (
	"testing"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestOutputConfigMissingBase(t *testing.T) {
	config := OutputConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestOutputConfigBuildValid(t *testing.T) {
	config := OutputConfig{
		BasicConfig: BasicConfig{
			OperatorID:   "test-id",
			OperatorType: "test-type",
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestOutputOperatorCanProcess(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputOperator{
		BasicOperator: BasicOperator{
			OperatorID:    "test-id",
			OperatorType:  "test-type",
			SugaredLogger: buildContext.Logger.SugaredLogger,
		},
	}
	require.True(t, output.CanProcess())
}

func TestOutputOperatorCanOutput(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputOperator{
		BasicOperator: BasicOperator{
			OperatorID:    "test-id",
			OperatorType:  "test-type",
			SugaredLogger: buildContext.Logger.SugaredLogger,
		},
	}
	require.False(t, output.CanOutput())
}

func TestOutputOperatorOutputs(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputOperator{
		BasicOperator: BasicOperator{
			OperatorID:    "test-id",
			OperatorType:  "test-type",
			SugaredLogger: buildContext.Logger.SugaredLogger,
		},
	}
	require.Equal(t, []operator.Operator{}, output.Outputs())
}

func TestOutputOperatorSetOutputs(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	output := OutputOperator{
		BasicOperator: BasicOperator{
			OperatorID:    "test-id",
			OperatorType:  "test-type",
			SugaredLogger: buildContext.Logger.SugaredLogger,
		},
	}

	err := output.SetOutputs([]operator.Operator{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operator can not output")
}
