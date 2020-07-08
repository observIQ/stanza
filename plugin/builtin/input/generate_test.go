package input

import (
	"testing"
	"text/template"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
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
				WriterConfig: helper.WriterConfig{
					OutputIDs: []string{"output1"},
				},
			},
			Entry: entry.Entry{
				Record: "test message",
			},
			Count: count,
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
    entry:
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
						WriterConfig: helper.WriterConfig{
							OutputIDs: []string{"sampleoutput"},
						},
					},
					Entry: entry.Entry{
						Record: map[interface{}]interface{}{
							"message": "testmessage",
						},
					},
				},
			},
		},
	}

	require.Equal(t, expectedConfig, config)
}
