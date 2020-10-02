package pipeline

import (
	"testing"

	"github.com/observiq/stanza-master/plugin"
	_ "github.com/observiq/stanza/operator/builtin/input/generate"
	_ "github.com/observiq/stanza/operator/builtin/transformer/noop"
	"github.com/stretchr/testify/require"
	// "github.com/observiq/stanza/plugin"
)

// type invalidMarshaller struct{}

// func (i invalidMarshaller) MarshalYAML() (interface{}, error) {
// 	return nil, fmt.Errorf("failed")
// }

func TestBuildValidPipeline(t *testing.T) {
	pluginTemplate := `
pipeline:
  - id: plugin_generate
    type: generate_input
    count: 1
    entry:
      record:
        message: test
    output: {{.output}}
`
	registry := plugin.Registry{}
	err := registry.Add("plugin", pluginTemplate)
	require.NoError(t, err)

	pipelineConfig := Config{
		Params{
			"id":     "plugin",
			"type":   "plugin",
			"output": "drop_output",
		},
		Params{
			"id":   "drop_output",
			"type": "drop_output",
		},
	}

	_, err = pipelineConfig.BuildPipeline(context, registry, nil)
	require.NoError(t, err)
}

// func TestBuildValidPipelineDefaultOutput(t *testing.T) {
// 	context := testutil.NewBuildContext(t)

// 	pipelineConfig := Config{
// 		Params{
// 			"id":    "generate_input",
// 			"type":  "generate_input",
// 			"count": 1,
// 			"entry": map[string]interface{}{
// 				"record": map[string]interface{}{
// 					"message": "test",
// 				},
// 			},
// 		},
// 	}

// 	defaultOutput, err := drop.NewDropOutputConfig("$.drop_it").Build(context)
// 	require.NoError(t, err)

// 	pl, err := pipelineConfig.BuildPipeline(context, nil, defaultOutput)
// 	require.NoError(t, err)
// 	require.True(t, pl.Graph.HasEdgeFromTo(createNodeID("$.generate_input"), createNodeID("$.drop_it")))
// }

// func TestBuildValidPipelineNextOutputAndDefaultOutput(t *testing.T) {
// 	context := testutil.NewBuildContext(t)

// 	pipelineConfig := Config{
// 		Params{
// 			"id":    "generate_input",
// 			"type":  "generate_input",
// 			"count": 1,
// 			"entry": map[string]interface{}{
// 				"record": map[string]interface{}{
// 					"message": "test",
// 				},
// 			},
// 		},
// 		Params{
// 			"id":   "noop",
// 			"type": "noop",
// 		},
// 	}

// 	defaultOutput, err := drop.NewDropOutputConfig("$.drop_it").Build(context)
// 	require.NoError(t, err)

// 	pl, err := pipelineConfig.BuildPipeline(context, nil, defaultOutput)
// 	require.NoError(t, err)
// 	require.True(t, pl.Graph.HasEdgeFromTo(createNodeID("$.generate_input"), createNodeID("$.noop")))
// 	require.True(t, pl.Graph.HasEdgeFromTo(createNodeID("$.noop"), createNodeID("$.drop_it")))
// }

// // func TestBuildValidPluginDefaultOutput(t *testing.T) {
// // 	context := testutil.NewBuildContext(t)
// // 	pluginTemplate := `
// // pipeline:
// //   - id: plugin_generate
// //     type: generate_input
// //     count: 1
// //     entry:
// //       record:
// //         message: test
// // `
// // 	registry := plugin.Registry{}
// // 	err := registry.Add("plugin", pluginTemplate)
// // 	require.NoError(t, err)

// // 	pipelineConfig := Config{
// // 		Params{
// // 			"id":   "plugin",
// // 			"type": "plugin",
// // 		},
// // 	}

// // 	defaultOutput, err := drop.NewDropOutputConfig("$.drop_it").Build(context)
// // 	require.NoError(t, err)

// // 	pl, err := pipelineConfig.BuildPipeline(context, registry, defaultOutput)
// // 	require.NoError(t, err)
// // 	require.True(t, pl.Graph.HasEdgeFromTo(createNodeID("$.plugin.plugin_generate"), createNodeID("$.drop_it")))
// // }

// func TestBuildInvalidPipelineInvalidType(t *testing.T) {
// 	context := testutil.NewBuildContext(t)

// 	pipelineConfig := Config{
// 		Params{
// 			"id":     "plugin",
// 			"type":   "plugin",
// 			"output": "drop_output",
// 		},
// 		Params{
// 			"id":   "drop_output",
// 			"type": "drop_output",
// 		},
// 	}

// 	_, err := pipelineConfig.BuildPipeline(context, nil, nil)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "unsupported `type` for operator config")
// }

// // func TestBuildInvalidPipelineInvalidParam(t *testing.T) {
// // 	context := testutil.NewBuildContext(t)
// // 	pluginTemplate := `
// // pipeline:
// //   - id: plugin_generate
// //     type: generate_input
// //     count: invalid_value
// //     record:
// //       message: test
// //     output: {{.output}}
// // `
// // 	registry := plugin.Registry{}
// // 	err := registry.Add("plugin", pluginTemplate)
// // 	require.NoError(t, err)

// // 	pipelineConfig := Config{
// // 		Params{
// // 			"id":     "plugin",
// // 			"type":   "plugin",
// // 			"output": "drop_output",
// // 		},
// // 		Params{
// // 			"id":   "drop_output",
// // 			"type": "drop_output",
// // 		},
// // 	}

// // 	_, err = pipelineConfig.BuildPipeline(context, registry, nil)
// // 	require.Error(t, err)
// // 	require.Contains(t, err.Error(), "build operator configs")
// // }

// func TestBuildInvalidPipelineInvalidOperator(t *testing.T) {
// 	pipelineConfig := Config{
// 		Params{
// 			"id":     "generate_input",
// 			"type":   "generate_input",
// 			"number": 1,
// 			"output": "drop_output",
// 		},
// 		Params{
// 			"id":   "drop_output",
// 			"type": "drop_output",
// 		},
// 	}

// 	context := testutil.NewBuildContext(t)
// 	_, err := pipelineConfig.BuildPipeline(context, nil, nil)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "field number not found")
// }

// func TestBuildInvalidPipelineInvalidGraph(t *testing.T) {
// 	pipelineConfig := Config{
// 		Params{
// 			"id":    "generate_input",
// 			"type":  "generate_input",
// 			"count": 1,
// 			"entry": map[string]interface{}{
// 				"record": map[string]interface{}{
// 					"message": "test",
// 				},
// 			},
// 			"output": "invalid_output",
// 		},
// 		Params{
// 			"id":   "drop_output",
// 			"type": "drop_output",
// 		},
// 	}

// 	context := testutil.NewBuildContext(t)
// 	_, err := pipelineConfig.BuildPipeline(context, nil, nil)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "does not exist")
// }

// // func TestBuildPipelineDefaultOutputInPlugin(t *testing.T) {
// // 	context := testutil.NewBuildContext(t)
// // 	pluginTemplate := `
// // pipeline:
// //   - id: plugin_generate1
// //     type: generate_input
// //     entry:
// //       record: test
// //     output: {{.output}}
// //   - id: plugin_generate2
// //     type: generate_input
// //     entry:
// //       record: test
// //     output: {{.output}}
// // `
// // 	registry := plugin.Registry{}
// // 	err := registry.Add("plugin", pluginTemplate)
// // 	require.NoError(t, err)

// // 	config := Config{
// // 		{
// // 			"id":   "my_plugin",
// // 			"type": "plugin",
// // 		},
// // 		{
// // 			"id":   "my_drop",
// // 			"type": "drop_output",
// // 		},
// // 	}

// // 	configs, err := config.buildOperatorConfigs("$", registry)
// // 	require.NoError(t, err)
// // 	require.Len(t, configs, 3)

// // 	operators, err := config.buildOperators(configs, context)
// // 	require.NoError(t, err)
// // 	require.Len(t, operators, 3)

// // 	for _, operator := range operators {
// // 		if !operator.CanOutput() {
// // 			continue
// // 		}
// // 		if err := operator.SetOutputs(operators); err != nil {
// // 			require.NoError(t, err)
// // 		}
// // 	}

// // 	require.Len(t, operators[0].Outputs(), 1)
// // 	require.Equal(t, "$.my_drop", operators[0].Outputs()[0].ID())
// // 	require.Len(t, operators[1].Outputs(), 1)
// // 	require.Equal(t, "$.my_drop", operators[1].Outputs()[0].ID())
// // 	require.Len(t, operators[2].Outputs(), 0)
// // }

// func TestMultiRoundtripParams(t *testing.T) {
// 	cases := []Params{
// 		map[string]interface{}{"foo": "bar"},
// 		map[string]interface{}{
// 			"foo": map[string]interface{}{
// 				"bar": "baz",
// 			},
// 		},
// 		map[string]interface{}{
// 			"123": map[string]interface{}{
// 				"234": "345",
// 			},
// 		},
// 		map[string]interface{}{
// 			"array": []string{
// 				"foo",
// 				"bar",
// 			},
// 		},
// 		map[string]interface{}{
// 			"array": []map[string]interface{}{
// 				{
// 					"foo": "bar",
// 				},
// 			},
// 		},
// 	}

// 	for _, tc := range cases {
// 		// To YAML
// 		marshalledYaml, err := yaml.Marshal(tc)
// 		require.NoError(t, err)

// 		// From YAML
// 		var unmarshalledYaml Params
// 		err = yaml.Unmarshal(marshalledYaml, &unmarshalledYaml)
// 		require.NoError(t, err)

// 		// To JSON
// 		marshalledJson, err := json.Marshal(unmarshalledYaml)
// 		require.NoError(t, err)

// 		// From JSON
// 		var unmarshalledJson Params
// 		err = json.Unmarshal(marshalledJson, &unmarshalledJson)
// 		require.NoError(t, err)

// 		// Back to YAML
// 		marshalledYaml2, err := yaml.Marshal(unmarshalledJson)
// 		require.NoError(t, err)
// 		require.Equal(t, marshalledYaml, marshalledYaml2)
// 	}
// }

// func TestBuildPipelineWithFailingOperator(t *testing.T) {
// 	ctx := testutil.NewBuildContext(t)

// 	type invalidOperatorConfig struct {
// 		OperatorType string `json:"type" yaml:"type"`
// 		testutil.OperatorBuilder
// 	}

// 	newBuilder := func() operator.Builder {
// 		config := &invalidOperatorConfig{}
// 		config.On("Build", mock.Anything).Return(nil, fmt.Errorf("failed to build operator"))
// 		config.On("ID").Return("test_id")
// 		config.On("Type").Return("invalid_operator")
// 		return config
// 	}

// 	operator.RegisterOperator("invalid_operator", newBuilder)
// 	config := Config{
// 		{"type": "invalid_operator"},
// 	}
// 	_, err := config.BuildPipeline(ctx, nil, nil)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "failed to build operator")
// }

// func TestBuildPipelineWithInvalidParam(t *testing.T) {
// 	ctx := testutil.NewBuildContext(t)
// 	config := Config{
// 		{"missing": "type"},
// 	}
// 	_, err := config.BuildPipeline(ctx, nil, nil)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "missing required `type` field")
// }

// type invalidYaml struct{}

// func (y invalidYaml) MarshalYAML() (interface{}, error) {
// 	return nil, fmt.Errorf("invalid yaml")
// }

// func TestBuildAsBuiltinWithInvalidParam(t *testing.T) {
// 	params := Params{
// 		"field": invalidYaml{},
// 	}
// 	_, err := params.buildAsBuiltin("test_namespace")
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "failed to parse config map as yaml")
// }

// func TestUnmarshalParamsWithInvalidBytes(t *testing.T) {
// 	bytes := []byte("string")
// 	var params Params
// 	err := yaml.Unmarshal(bytes, &params)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "unmarshal errors")
// }

// func TestCleanValueWithUnknownType(t *testing.T) {
// 	value := cleanValue(map[int]int{})
// 	require.Equal(t, "map[]", value)
// }
