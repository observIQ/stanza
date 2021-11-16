package agent

import (
	"fmt"
	"testing"

	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartAgentSuccess(t *testing.T) {
	logger := zap.NewNop().Sugar()
	pipeline := &testutil.Pipeline{}
	pipeline.On("Start").Return(nil)

	agent := LogAgent{
		SugaredLogger: logger,
		pipeline:      pipeline,
	}
	err := agent.Start()
	require.NoError(t, err)
	pipeline.AssertCalled(t, "Start")
}

func TestStartAgentFailure(t *testing.T) {
	logger := zap.NewNop().Sugar()
	pipeline := &testutil.Pipeline{}
	failure := fmt.Errorf("failed to start pipeline")
	pipeline.On("Start").Return(failure)

	agent := LogAgent{
		SugaredLogger: logger,
		pipeline:      pipeline,
	}
	err := agent.Start()
	require.Error(t, err, failure)
	pipeline.AssertCalled(t, "Start")
}

func TestStopAgentSuccess(t *testing.T) {
	logger := zap.NewNop().Sugar()
	pipeline := &testutil.Pipeline{}
	pipeline.On("Stop").Return(nil)
	database := &testutil.Database{}
	database.On("Close").Return(nil)

	agent := LogAgent{
		SugaredLogger: logger,
		pipeline:      pipeline,
		database:      database,
	}
	err := agent.Stop()
	require.NoError(t, err)
	pipeline.AssertCalled(t, "Stop")
	database.AssertCalled(t, "Close")
}

func TestStopAgentPipelineFailure(t *testing.T) {
	logger := zap.NewNop().Sugar()
	pipeline := &testutil.Pipeline{}
	failure := fmt.Errorf("failed to start pipeline")
	pipeline.On("Stop").Return(failure)
	database := &testutil.Database{}
	database.On("Close").Return(nil)

	agent := LogAgent{
		SugaredLogger: logger,
		pipeline:      pipeline,
		database:      database,
	}
	err := agent.Stop()
	require.Error(t, err, failure)
	pipeline.AssertCalled(t, "Stop")
	database.AssertNotCalled(t, "Close")
}

func TestStopAgentDatabaseFailure(t *testing.T) {
	logger := zap.NewNop().Sugar()
	pipeline := &testutil.Pipeline{}
	pipeline.On("Stop").Return(nil)
	database := &testutil.Database{}
	failure := fmt.Errorf("failed to close database")
	database.On("Close").Return(failure)

	agent := LogAgent{
		SugaredLogger: logger,
		pipeline:      pipeline,
		database:      database,
	}
	err := agent.Stop()
	require.Error(t, err, failure)
	pipeline.AssertCalled(t, "Stop")
	database.AssertCalled(t, "Close")
}
