package agent

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildAgentSuccess(t *testing.T) {
	mockCfg := Config{}
	mockLogger := zap.NewNop().Sugar()
	mockPluginDir := "/some/path/plugins"
	mockDatabaseFile := ""
	mockOutput := testutil.NewFakeOutput(t)

	agent, err := NewBuilder(&mockCfg, mockLogger).
		WithPluginDir(mockPluginDir).
		WithDatabaseFile(mockDatabaseFile).
		WithDefaultOutput(mockOutput).
		Build()
	require.NoError(t, err)
	require.Equal(t, mockLogger, agent.SugaredLogger)
}

func TestBuildAgentFailureOnDatabase(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	invalidDatabaseFile := filepath.Join(tempDir, "test.db")
	err := ioutil.WriteFile(invalidDatabaseFile, []byte("invalid"), 0755)
	require.NoError(t, err)

	mockCfg := Config{}
	mockLogger := zap.NewNop().Sugar()
	mockPluginDir := "/some/path/plugins"
	mockDatabaseFile := invalidDatabaseFile
	mockOutput := testutil.NewFakeOutput(t)

	agent, err := NewBuilder(&mockCfg, mockLogger).
		WithPluginDir(mockPluginDir).
		WithDatabaseFile(mockDatabaseFile).
		WithDefaultOutput(mockOutput).
		Build()
	require.Error(t, err)
	require.Nil(t, agent)
}

func TestBuildAgentFailureOnPluginRegistry(t *testing.T) {
	mockCfg := Config{}
	mockLogger := zap.NewNop().Sugar()
	mockPluginDir := "[]"
	mockDatabaseFile := ""
	mockOutput := testutil.NewFakeOutput(t)

	agent, err := NewBuilder(&mockCfg, mockLogger).
		WithPluginDir(mockPluginDir).
		WithDatabaseFile(mockDatabaseFile).
		WithDefaultOutput(mockOutput).
		Build()
	require.Error(t, err)
	require.Nil(t, agent)
}

// TODO
// func TestBuildAgentFailureOnPipeline(t *testing.T) {
// 	mockCfg := Config{
// 		Pipeline: pipeline.Config{
// 			pipeline.Params{"type": "missing"},
// 		},
// 	}
// 	mockLogger := zap.NewNop().Sugar()
// 	mockPluginDir := "/some/path/plugins"
// 	mockDatabaseFile := ""
// 	mockOutput := testutil.NewFakeOutput(t)

// 	agent, err := NewBuilder(&mockCfg, mockLogger).
// 		WithPluginDir(mockPluginDir).
// 		WithDatabaseFile(mockDatabaseFile).
// 		WithDefaultOutput(mockOutput).
// 		Build()
// 	require.Error(t, err)
// 	require.Nil(t, agent)
// }
