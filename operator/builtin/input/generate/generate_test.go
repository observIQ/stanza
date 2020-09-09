package generate

import (
	"testing"
	"text/template"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/plugin"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInputGenerate(t *testing.T) {
	count := 5
	basicConfig := func() *GenerateInputConfig {
		cfg := NewGenerateInputConfig("test_operator_id")
		cfg.OutputIDs = []string{"output1"}
		cfg.Entry = entry.Entry{
			Record: "test message",
		}
		cfg.Count = count
		return cfg
	}

	buildContext := testutil.NewBuildContext(t)
	newOperator, err := basicConfig().Build(buildContext)
	require.NoError(t, err)

	receivedEntries := make(chan *entry.Entry)
	mockOutput := testutil.NewMockOperator("output1")
	mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		receivedEntries <- args.Get(1).(*entry.Entry)
	})

	generateInput := newOperator.(*GenerateInput)
	err = generateInput.SetOutputs([]operator.Operator{mockOutput})
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

func TestRenderFromPluginTemplate(t *testing.T) {
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

	registry := plugin.Registry{
		"sample": tmpl,
	}

	params := map[string]interface{}{
		"output": "sampleoutput",
	}
	config, err := registry.Render("sample", params)
	require.NoError(t, err)

	expectedConfig := plugin.Plugin{
		Pipeline: []operator.Config{
			{
				Builder: &GenerateInputConfig{
					InputConfig: helper.InputConfig{
						LabelerConfig:    helper.NewLabelerConfig(),
						IdentifierConfig: helper.NewIdentifierConfig(),
						WriteTo:          entry.NewRecordField(),
						WriterConfig: helper.WriterConfig{
							BasicConfig: helper.BasicConfig{
								OperatorID:   "my_generator",
								OperatorType: "generate_input",
							},
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
