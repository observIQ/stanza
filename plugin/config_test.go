package plugin

import (
	"testing"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/transformer/noop"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/pipeline"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestGetRenderParams(t *testing.T) {
	cfg := Config{}
	cfg.OperatorID = "test"
	cfg.Parameters = map[string]interface{}{
		"param1": "value1",
		"param2": "value2",
	}
	cfg.OutputIDs = []string{"out1", "out2"}

	params := cfg.getRenderParams(testutil.NewBuildContext(t))
	expected := map[string]interface{}{
		"param1": "value1",
		"param2": "value2",
		"input":  "$.test",
		"output": "[$.out1,$.out2]",
	}
	require.Equal(t, expected, params)
}

func TestPlugin(t *testing.T) {
	pluginContent := []byte(`
parameters:
pipeline:
  - id: {{ .input }}
    type: noop
    output: {{ .output }}
`)

	configContent := []byte(`
id: my_plugin_id
type: my_plugin
unused_param: test_unused
output: stdout
`)

	plugin, err := NewPlugin("my_plugin", pluginContent)
	require.NoError(t, err)

	operator.RegisterPlugin("my_plugin", plugin.NewBuilder)

	var cfg operator.Config
	err = yaml.Unmarshal(configContent, &cfg)
	require.NoError(t, err)

	expected := operator.Config{
		MultiBuilder: &Config{
			WriterConfig: helper.WriterConfig{
				OutputIDs: []string{"stdout"},
				BasicConfig: helper.BasicConfig{
					OperatorID:   "my_plugin_id",
					OperatorType: "my_plugin",
				},
			},
			plugin: plugin,
			Parameters: map[string]interface{}{
				"unused_param": "test_unused",
			},
		},
	}

	require.Equal(t, expected, cfg)

	operators, err := cfg.BuildMulti(testutil.NewBuildContext(t))
	require.Len(t, operators, 1)
	noop, ok := operators[0].(*noop.NoopOperator)
	require.True(t, ok)
	require.Equal(t, "send", noop.OnError)
	require.Equal(t, "$.my_plugin_id", noop.OperatorID)
	require.Equal(t, "noop", noop.OperatorType)
}

func TestBuildRecursiveFails(t *testing.T) {
	pluginConfig1 := []byte(`
pipeline:
  - type: plugin2
`)

	pluginConfig2 := []byte(`
pipeline:
  - type: plugin1
`)

	plugin1, err := NewPlugin("plugin1", pluginConfig1)
	require.NoError(t, err)
	plugin2, err := NewPlugin("plugin2", pluginConfig2)
	require.NoError(t, err)

	t.Cleanup(func() { operator.DefaultRegistry = operator.NewRegistry() })
	operator.RegisterPlugin("plugin1", plugin1.NewBuilder)
	operator.RegisterPlugin("plugin2", plugin2.NewBuilder)

	pipelineConfig := []byte(`
- type: plugin1
`)

	var pipeline pipeline.Config
	err = yaml.Unmarshal(pipelineConfig, &pipeline)
	require.NoError(t, err)

	_, err = pipeline.BuildOperators(operator.NewBuildContext(nil, nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "reached max plugin depth")
}
