package helper

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestWriterConfigMissingOutput(t *testing.T) {
	config := WriterConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `output` field")
}

func TestWriterConfigValidBuild(t *testing.T) {
	config := WriterConfig{
		OutputIDs: OutputIDs{"output"},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestWriterConfigSetNamespace(t *testing.T) {
	config := WriterConfig{
		OutputIDs: OutputIDs{"output1", "output2"},
	}
	config.SetNamespace("namespace")
	require.Equal(t, OutputIDs{"namespace.output1", "namespace.output2"}, config.OutputIDs)
}

func TestWriterPluginWrite(t *testing.T) {
	output1 := &testutil.Plugin{}
	output1.On("Process", mock.Anything, mock.Anything).Return(nil)
	output2 := &testutil.Plugin{}
	output2.On("Process", mock.Anything, mock.Anything).Return(nil)
	writer := WriterPlugin{
		OutputPlugins: []plugin.Plugin{output1, output2},
	}

	ctx := context.Background()
	testEntry := entry.New()

	writer.Write(ctx, testEntry)
	output1.AssertCalled(t, "Process", ctx, testEntry)
	output2.AssertCalled(t, "Process", ctx, testEntry)
}

func TestWriterPluginCanOutput(t *testing.T) {
	writer := WriterPlugin{}
	require.True(t, writer.CanOutput())
}

func TestWriterPluginOutputs(t *testing.T) {
	output1 := &testutil.Plugin{}
	output1.On("Process", mock.Anything, mock.Anything).Return(nil)
	output2 := &testutil.Plugin{}
	output2.On("Process", mock.Anything, mock.Anything).Return(nil)
	writer := WriterPlugin{
		OutputPlugins: []plugin.Plugin{output1, output2},
	}

	ctx := context.Background()
	testEntry := entry.New()

	writer.Write(ctx, testEntry)
	output1.AssertCalled(t, "Process", ctx, testEntry)
	output2.AssertCalled(t, "Process", ctx, testEntry)
}

func TestWriterSetOutputsMissing(t *testing.T) {
	output1 := &testutil.Plugin{}
	output1.On("ID").Return("output1")
	writer := WriterPlugin{
		OutputIDs: OutputIDs{"output2"},
	}

	err := writer.SetOutputs([]plugin.Plugin{output1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestWriterSetOutputsInvalid(t *testing.T) {
	output1 := &testutil.Plugin{}
	output1.On("ID").Return("output1")
	output1.On("CanProcess").Return(false)
	writer := WriterPlugin{
		OutputIDs: OutputIDs{"output1"},
	}

	err := writer.SetOutputs([]plugin.Plugin{output1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can not process entries")
}

func TestWriterSetOutputsValid(t *testing.T) {
	output1 := &testutil.Plugin{}
	output1.On("ID").Return("output1")
	output1.On("CanProcess").Return(true)
	output2 := &testutil.Plugin{}
	output2.On("ID").Return("output2")
	output2.On("CanProcess").Return(true)
	writer := WriterPlugin{
		OutputIDs: OutputIDs{"output1", "output2"},
	}

	err := writer.SetOutputs([]plugin.Plugin{output1, output2})
	require.NoError(t, err)
	require.Equal(t, []plugin.Plugin{output1, output2}, writer.Outputs())
}

func TestUnmarshalJSONString(t *testing.T) {
	bytes := []byte("{\"output\":\"test\"}")
	var config WriterConfig
	err := json.Unmarshal(bytes, &config)
	require.NoError(t, err)
	require.Equal(t, OutputIDs{"test"}, config.OutputIDs)
}

func TestUnmarshalJSONArray(t *testing.T) {
	bytes := []byte("{\"output\":[\"test1\",\"test2\"]}")
	var config WriterConfig
	err := json.Unmarshal(bytes, &config)
	require.NoError(t, err)
	require.Equal(t, OutputIDs{"test1", "test2"}, config.OutputIDs)
}

func TestUnmarshalJSONInvalidValue(t *testing.T) {
	bytes := []byte("{\"output\": true}")
	var config WriterConfig
	err := json.Unmarshal(bytes, &config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "value is not of type string or string array")
}

func TestUnmarshalJSONInvalidArray(t *testing.T) {
	bytes := []byte("{\"output\":[\"test1\", true]}")
	var config WriterConfig
	err := json.Unmarshal(bytes, &config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "value in array is not of type string")
}

func TestUnmarshalYAMLString(t *testing.T) {
	bytes := []byte("output: test")
	var config WriterConfig
	err := yaml.Unmarshal(bytes, &config)
	require.NoError(t, err)
	require.Equal(t, OutputIDs{"test"}, config.OutputIDs)
}

func TestUnmarshalYAMLArray(t *testing.T) {
	bytes := []byte("output: [test1, test2]")
	var config WriterConfig
	err := yaml.Unmarshal(bytes, &config)
	require.NoError(t, err)
	require.Equal(t, OutputIDs{"test1", "test2"}, config.OutputIDs)
}

func TestUnmarshalYAMLInvalidValue(t *testing.T) {
	bytes := []byte("output: true")
	var config WriterConfig
	err := yaml.Unmarshal(bytes, &config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "value is not of type string or string array")
}

func TestUnmarshalYAMLInvalidArray(t *testing.T) {
	bytes := []byte("output: [test1, true]")
	var config WriterConfig
	err := yaml.Unmarshal(bytes, &config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "value in array is not of type string")
}
