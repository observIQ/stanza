package input

import (
	"testing"
	"text/template"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInputGenerate(t *testing.T) {
	count := 5
	basicConfig := func() *GenerateInputConfig {
		return &GenerateInputConfig{
			InputConfig: helper.InputConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "test_plugin_id",
					PluginType: "generate_input",
				},
				WriteTo: entry.Field{
					Keys: []string{},
				},
				OutputID: "output1",
			},
			Record: "test message",
			Count:  count,
		}
	}

	buildContext := testutil.NewBuildContext(t)
	newPlugin, err := basicConfig().Build(buildContext)
	require.NoError(t, err)

	receivedEntries := make(chan *entry.Entry)
	mockOutput := testutil.NewMockPlugin("output1")
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedEntries <- args.Get(1).(*entry.Entry)
	})

	generateInput := newPlugin.(*GenerateInput)
	err = generateInput.SetOutputs([]plugin.Plugin{mockOutput})
	require.NoError(t, err)

	err = generateInput.Start()
	require.NoError(t, err)
	defer generateInput.Stop()

	for i := 0; i < count; i++ {
		select {
		case <-receivedEntries:
			continue
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for generated entries")
		}
	}
}

func TestRenderFromCustom(t *testing.T) {
	templateText := `
pipeline:
  - id: my_generator
    type: generate_input
    output: {{ .output }}
    record:
      message: testmessage
`
	tmpl, err := template.New("my_generator").Parse(templateText)
	require.NoError(t, err)

	registry := plugin.CustomRegistry{
		"sample": tmpl,
	}

	params := map[string]interface{}{
		"output": "sampleoutput",
	}
	config, err := registry.Render("sample", params)
	require.NoError(t, err)

	expectedConfig := plugin.CustomConfig{
		Pipeline: []plugin.Config{
			{
				Builder: &GenerateInputConfig{
					InputConfig: helper.InputConfig{
						BasicConfig: helper.BasicConfig{
							PluginID:   "my_generator",
							PluginType: "generate_input",
						},
						OutputID: "sampleoutput",
					},
					Record: map[interface{}]interface{}{
						"message": "testmessage",
					},
				},
			},
		},
	}

	require.Equal(t, expectedConfig, config)
}
