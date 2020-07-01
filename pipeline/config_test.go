package pipeline

import (
	"fmt"
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	_ "github.com/bluemedora/bplogagent/plugin/builtin"
	"github.com/bluemedora/bplogagent/plugin/builtin/transformer"
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

func TestParamsWithOutputs(t *testing.T) {
	params := Params{
		"output": "test",
	}
	actualOutput := params.Outputs()
	require.Equal(t, []string{"test"}, actualOutput)
}

func TestParamsWithoutOutputs(t *testing.T) {
	params := Params{}
	actualOutput := params.Outputs()
	require.Equal(t, []string{}, actualOutput)
}

func TestParamsNamespacedID(t *testing.T) {
	params := Params{
		"id": "test-id",
	}
	result := params.NamespacedID("namespace")
	require.Equal(t, "namespace.test-id", result)
}

func TestParamsNamespacedOutputs(t *testing.T) {
	params := Params{
		"output": "test-output",
	}
	result := params.NamespacedOutputs("namespace")
	require.Equal(t, []string{"namespace.test-output"}, result)
}

func TestParamsTemplateInput(t *testing.T) {
	params := Params{
		"id": "test-id",
	}
	result := params.TemplateInput("namespace")
	require.Equal(t, "namespace.test-id", result)
}

func TestParamsTemplateOutput(t *testing.T) {
	params := Params{
		"output": "test-output",
	}
	result := params.TemplateOutput("namespace")
	require.Equal(t, "[namespace.test-output]", result)
}

func TestParamsNamespaceExclusions(t *testing.T) {
	params := Params{
		"id":     "test-id",
		"output": "test-output",
	}
	result := params.NamespaceExclusions("namespace")
	require.Equal(t, []string{"namespace.test-id", "namespace.test-output"}, result)
}

func TestParamsGetExistingString(t *testing.T) {
	params := Params{
		"key": "string",
	}
	result := params.getString("key")
	require.Equal(t, "string", result)
}

func TestParamsGetMissingString(t *testing.T) {
	params := Params{}
	result := params.getString("missing")
	require.Equal(t, "", result)
}

func TestParamsGetInvalidString(t *testing.T) {
	params := Params{
		"key": true,
	}
	result := params.getString("key")
	require.Equal(t, "", result)
}

func TestParamsGetStringArrayMissing(t *testing.T) {
	params := Params{}
	result := params.getStringArray("missing")
	require.Equal(t, []string{}, result)
}

func TestParamsGetStringArrayFromString(t *testing.T) {
	params := Params{
		"key": "string",
	}
	result := params.getStringArray("key")
	require.Equal(t, []string{"string"}, result)
}

func TestParamsGetStringArrayFromArray(t *testing.T) {
	params := Params{
		"key": []string{"one", "two"},
	}
	result := params.getStringArray("key")
	require.Equal(t, []string{"one", "two"}, result)
}

func TestParamsGetStringArrayFromInterface(t *testing.T) {
	params := Params{
		"key": []interface{}{"one", "two"},
	}
	result := params.getStringArray("key")
	require.Equal(t, []string{"one", "two"}, result)
}

func TestParamsGetStringArrayFromInvalid(t *testing.T) {
	params := Params{
		"key": true,
	}
	result := params.getStringArray("key")
	require.Equal(t, []string{}, result)
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

type invalidMarshaller struct {}
func (i invalidMarshaller) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("failed")
}

func TestBuildBuiltinFromParamsWithUnsupportedYaml(t *testing.T) {
	params := Params{
		"id":     "noop",
		"type":   "noop",
		"output": "test",
		"field": invalidMarshaller{},
	}
	context := plugin.BuildContext{}
	_, err := params.BuildConfigs(context, "test_namespace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse config map as yaml")
}

func TestBuildBuiltinFromParamsWithUnknownField(t *testing.T) {
	params := Params{
		"id":      "noop",
		"type":    "noop",
		"unknown": true,
		"output":  "test_output",
	}
	context := plugin.BuildContext{}
	_, err := params.BuildConfigs(context, "test_namespace")
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
	require.IsType(t, &transformer.NoopPluginConfig{}, configs[0].Builder)
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
	require.IsType(t, &transformer.NoopPluginConfig{}, configs[0].Builder)
	require.Equal(t, "test_namespace.custom_plugin.custom_noop", configs[0].ID())
}

func TestBuildValidPipeline(t *testing.T) {
	registry := plugin.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_generate
    type: generate_input
    count: 1
    entry:
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

func TestBuildInvalidPipelineInvalidGraph(t *testing.T) {
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
			"entry": map[string]interface{}{
				"record": map[string]interface{}{
					"message": "test",
				},
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
