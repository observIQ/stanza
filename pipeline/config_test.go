package pipeline

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/builtin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParamsWithID(t *testing.T) {
	expectedID := "test"
	params := Params{
		"id": expectedID,
	}
	actualID := params.ID()
	require.Equal(t, expectedID, actualID)
}

func TestParamsWithoutID(t *testing.T) {
	params := Params{}
	actualID := params.ID()
	require.Equal(t, "", actualID)
}

func TestParamsWithType(t *testing.T) {
	expectedType := "test"
	params := Params{
		"type": expectedType,
	}
	actualType := params.Type()
	require.Equal(t, expectedType, actualType)
}

func TestParamsWithoutType(t *testing.T) {
	params := Params{}
	actualType := params.Type()
	require.Equal(t, "", actualType)
}

func TestParamsWithOutput(t *testing.T) {
	expectedOutput := "test"
	params := Params{
		"output": expectedOutput,
	}
	actualOutput := params.Output()
	require.Equal(t, expectedOutput, actualOutput)
}

func TestParamsWithoutOutput(t *testing.T) {
	params := Params{}
	actualOutput := params.Output()
	require.Equal(t, "", actualOutput)
}

func TestValidParams(t *testing.T) {
	params := Params{
		"id":   "test_id",
		"type": "test_type",
	}
	err := params.Validate()
	require.NoError(t, err)
}

func TestInvalidParams(t *testing.T) {
	paramsWithoutID := Params{
		"type": "test_type",
	}
	err := paramsWithoutID.Validate()
	require.Error(t, err)

	paramsWithoutType := Params{
		"id": "test_id",
	}
	err = paramsWithoutType.Validate()
	require.Error(t, err)
}

func TestBuildBuiltinFromValidParams(t *testing.T) {
	params := Params{
		"id":     "noop",
		"type":   "noop",
		"output": "test_output",
	}
	context := plugin.BuildContext{}
	configs, err := params.BuildConfigs(context, "test_namespace")

	require.NoError(t, err)
	require.Equal(t, 1, len(configs))
	require.IsType(t, &builtin.NoopPluginConfig{}, configs[0].Builder)
	require.Equal(t, "test_namespace.noop", configs[0].ID())
}

func TestBuildCustomFromValidParams(t *testing.T) {
	registry := plugin.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_noop
    type: noop
    output: {{.output}} 
`
	err := registry.Add("custom_plugin", customTemplate)
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
	}
	params := Params{
		"id":     "custom_plugin",
		"type":   "custom_plugin",
		"output": "test_output",
	}

	configs, err := params.BuildConfigs(context, "test_namespace")
	require.NoError(t, err)
	require.Equal(t, 1, len(configs))
	require.IsType(t, &builtin.NoopPluginConfig{}, configs[0].Builder)
	require.Equal(t, "test_namespace.custom_plugin.custom_noop", configs[0].ID())
}

func TestBuildValidPipeline(t *testing.T) {
	registry := plugin.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_generate
    type: generate_input
    count: 1
    record:
      message: test
    output: {{.output}} 
`
	err := registry.Add("custom_plugin", customTemplate)
	require.NoError(t, err)

	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_plugin",
			"type":   "custom_plugin",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.NoError(t, err)
}

func TestBuildInvalidPipelineInvalidType(t *testing.T) {
	registry := plugin.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_plugin",
			"type":   "custom_plugin",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported `type` for plugin config")
}

func TestBuildInvalidPipelineInvalidParam(t *testing.T) {
	registry := plugin.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_generate
    type: generate_input
    count: invalid_value
    record:
      message: test
    output: {{.output}} 
`
	err := registry.Add("custom_plugin", customTemplate)
	require.NoError(t, err)

	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_plugin",
			"type":   "custom_plugin",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build plugin configs")
}

func TestBuildInvalidPipelineInvalidPlugin(t *testing.T) {
	registry := plugin.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "tcp_input",
			"type":   "tcp_input",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build plugins")
}

func TestBuildInvalidPipelineInvalidConnection(t *testing.T) {
	registry := plugin.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := plugin.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":    "generate_input",
			"type":  "generate_input",
			"count": 1,
			"record": map[string]interface{}{
				"message": "test",
			},
			"output": "invalid_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "new pipeline")
}
