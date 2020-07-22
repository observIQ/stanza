package pipeline

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/observiq/carbon/operator"
	_ "github.com/observiq/carbon/operator/builtin"
	"github.com/observiq/carbon/operator/builtin/transformer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
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
	paramsWithoutType := Params{
		"id": "test_id",
	}
	err := paramsWithoutType.Validate()
	require.Error(t, err)
}

type invalidMarshaller struct{}

func (i invalidMarshaller) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("failed")
}

func TestBuildBuiltinFromParamsWithUnsupportedYaml(t *testing.T) {
	params := Params{
		"id":     "noop",
		"type":   "noop",
		"output": "test",
		"field":  invalidMarshaller{},
	}
	_, err := params.BuildConfigs(operator.CustomRegistry{}, "test_namespace")
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
	_, err := params.BuildConfigs(operator.CustomRegistry{}, "test_namespace")
	require.Error(t, err)
}

func TestBuildBuiltinFromValidParams(t *testing.T) {
	params := Params{
		"id":     "noop",
		"type":   "noop",
		"output": "test_output",
	}
	configs, err := params.BuildConfigs(operator.CustomRegistry{}, "test_namespace")

	require.NoError(t, err)
	require.Equal(t, 1, len(configs))
	require.IsType(t, &transformer.NoopOperatorConfig{}, configs[0].Builder)
	require.Equal(t, "test_namespace.noop", configs[0].ID())
}

func TestBuildCustomFromValidParams(t *testing.T) {
	registry := operator.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_noop
    type: noop
    output: {{.output}}
`
	err := registry.Add("custom_operator", customTemplate)
	require.NoError(t, err)

	params := Params{
		"id":     "custom_operator",
		"type":   "custom_operator",
		"output": "test_output",
	}

	configs, err := params.BuildConfigs(registry, "test_namespace")
	require.NoError(t, err)
	require.Equal(t, 1, len(configs))
	require.IsType(t, &transformer.NoopOperatorConfig{}, configs[0].Builder)
	require.Equal(t, "test_namespace.custom_operator.custom_noop", configs[0].ID())
}

func TestBuildValidPipeline(t *testing.T) {
	registry := operator.CustomRegistry{}
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
	err := registry.Add("custom_operator", customTemplate)
	require.NoError(t, err)

	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := operator.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_operator",
			"type":   "custom_operator",
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
	registry := operator.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := operator.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_operator",
			"type":   "custom_operator",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported `type` for operator config")
}

func TestBuildInvalidPipelineInvalidParam(t *testing.T) {
	registry := operator.CustomRegistry{}
	customTemplate := `
pipeline:
  - id: custom_generate
    type: generate_input
    count: invalid_value
    record:
      message: test
    output: {{.output}}
`
	err := registry.Add("custom_operator", customTemplate)
	require.NoError(t, err)

	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := operator.BuildContext{
		CustomRegistry: registry,
		Logger:         logger.Sugar(),
	}

	pipelineConfig := Config{
		Params{
			"id":     "custom_operator",
			"type":   "custom_operator",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build operator configs")
}

func TestBuildInvalidPipelineInvalidOperator(t *testing.T) {
	registry := operator.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := operator.BuildContext{
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
	require.Contains(t, err.Error(), "missing required parameter 'listen_address'")
}

func TestBuildInvalidPipelineInvalidGraph(t *testing.T) {
	registry := operator.CustomRegistry{}
	logCfg := zap.NewProductionConfig()
	logger, err := logCfg.Build()
	require.NoError(t, err)

	context := operator.BuildContext{
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
	require.Contains(t, err.Error(), "does not exist")
}

func TestMultiRoundtripParams(t *testing.T) {
	cases := []Params{
		map[string]interface{}{"foo": "bar"},
		map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
		},
		map[string]interface{}{
			"123": map[string]interface{}{
				"234": "345",
			},
		},
		map[string]interface{}{
			"array": []string{
				"foo",
				"bar",
			},
		},
		map[string]interface{}{
			"array": []map[string]interface{}{
				{
					"foo": "bar",
				},
			},
		},
	}

	for _, tc := range cases {
		// To YAML
		marshalledYaml, err := yaml.Marshal(tc)
		require.NoError(t, err)

		// From YAML
		var unmarshalledYaml Params
		err = yaml.Unmarshal(marshalledYaml, &unmarshalledYaml)
		require.NoError(t, err)

		// To JSON
		marshalledJson, err := json.Marshal(unmarshalledYaml)
		require.NoError(t, err)

		// From JSON
		var unmarshalledJson Params
		err = json.Unmarshal(marshalledJson, &unmarshalledJson)
		require.NoError(t, err)

		// Back to YAML
		marshalledYaml2, err := yaml.Marshal(unmarshalledJson)
		require.NoError(t, err)
		require.Equal(t, marshalledYaml, marshalledYaml2)
	}
}
