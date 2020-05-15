package config

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/builtin"
	"github.com/bluemedora/bplogagent/plugin/builtin/fileinput"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

var testRepresentativeYAML = []byte(`
plugins:
  - id: my_file_input
    type: file_input
    output: my_restructure
    write_to: message
    include:
      - "./testfile"
  - id: my_restructure
    type: restructure
    output: my_logger
    ops:
      - add:
          field: "message.nested"
          value: "testvalue"
      - add:
          field: "message.nested2"
          value: "testvalue2"
      - remove: "message.nested2"
      - move:
          from: "message.nested"
          to: "message.nested3"
      - retain:
        - "message.nested3"
      - flatten: "message"
  - id: my_logger
    type: logger_output
`)

var testRepresentativeJSON = []byte(`
{
  "plugins": [
    {
      "id": "my_file_input",
      "type": "file_input",
			"include": ["./testfile"],
			"write_to": "message",
      "output": "my_restructure"
    },
    {
      "id": "my_restructure",
      "type": "restructure",
      "output": "my_logger",
      "ops": [
        {
          "add": {
            "field": "message.nested",
            "value": "testvalue"
          }
        },
        {
          "add": {
            "field": "message.nested2",
            "value": "testvalue2"
          }
        },
        {
          "remove": "message.nested2"
        },
        {
          "move": {
            "from": "message.nested",
            "to": "message.nested3"
          }
        },
        {
          "retain": [
						"message.nested3"
          ]
        },
        {
          "flatten": "message"
        }
      ]
    },
    {
      "id": "my_logger",
      "type": "logger_output"
    }
  ]
}
`)

var testParsedRepresentativeConfig = Config{
	Plugins: []plugin.Config{
		{
			PluginBuilder: &fileinput.FileInputConfig{
				BasicPluginConfig: helper.BasicPluginConfig{
					PluginID:   "my_file_input",
					PluginType: "file_input",
				},
				BasicInputConfig: helper.BasicInputConfig{
					OutputID: "my_restructure",
					WriteTo:  entry.Field([]string{"message"}),
				},
				Include: []string{"./testfile"},
			},
		},
		{
			PluginBuilder: &builtin.RestructurePluginConfig{
				BasicPluginConfig: helper.BasicPluginConfig{
					PluginID:   "my_restructure",
					PluginType: "restructure",
				},
				BasicTransformerConfig: helper.BasicTransformerConfig{
					OutputID: "my_logger",
				},
				Ops: []builtin.Op{
					{
						OpApplier: &builtin.OpAdd{
							Field: entry.Field([]string{"message", "nested"}),
							Value: "testvalue",
						},
					},
					{
						OpApplier: &builtin.OpAdd{
							Field: entry.Field([]string{"message", "nested2"}),
							Value: "testvalue2",
						},
					},
					{
						OpApplier: &builtin.OpRemove{
							Field: entry.Field([]string{"message", "nested2"}),
						},
					},
					{
						OpApplier: &builtin.OpMove{
							From: entry.Field([]string{"message", "nested"}),
							To:   entry.Field([]string{"message", "nested3"}),
						},
					},
					{
						OpApplier: &builtin.OpRetain{
							Fields: []entry.Field{[]string{"message", "nested3"}},
						},
					},
					{
						OpApplier: &builtin.OpFlatten{
							Field: entry.Field([]string{"message"}),
						},
					},
				},
			},
		},
		{
			PluginBuilder: &builtin.LoggerOutputConfig{
				BasicPluginConfig: helper.BasicPluginConfig{
					PluginID:   "my_logger",
					PluginType: "logger_output",
				},
			},
		},
	},
}

func TestUnmarshalRepresentativeConfig(t *testing.T) {

	var mapConfig map[string]interface{}
	err := yaml.Unmarshal(testRepresentativeYAML, &mapConfig)
	require.NoError(t, err)

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:     &cfg,
		DecodeHook: DecodeHookFunc,
	})
	require.NoError(t, err)

	err = decoder.Decode(mapConfig)
	require.NoError(t, err)

	require.Equal(t, testParsedRepresentativeConfig, cfg)
}

func TestUnmarshalRepresentativeConfigYAML(t *testing.T) {
	var cfg Config
	err := yaml.Unmarshal(testRepresentativeYAML, &cfg)
	require.NoError(t, err)

	require.Equal(t, testParsedRepresentativeConfig, cfg)
}

func TestUnmarshalRepresentativeConfigJSON(t *testing.T) {
	var cfg Config
	err := json.Unmarshal(testRepresentativeJSON, &cfg)
	require.NoError(t, err)

	require.Equal(t, testParsedRepresentativeConfig, cfg)
}

func TestRoundTripRepresentativeConfigYAML(t *testing.T) {
	marshalled, err := yaml.Marshal(testParsedRepresentativeConfig)
	require.NoError(t, err)

	fmt.Print(string(marshalled))

	var cfg Config
	err = yaml.Unmarshal(marshalled, &cfg)
	require.NoError(t, err)

	require.Equal(t, testParsedRepresentativeConfig, cfg)
}

func TestRoundTripRepresentativeConfigJSON(t *testing.T) {
	marshalled, err := json.Marshal(testParsedRepresentativeConfig)
	require.NoError(t, err)

	var cfg Config
	err = json.Unmarshal(marshalled, &cfg)
	require.NoError(t, err)

	require.Equal(t, testParsedRepresentativeConfig, cfg)

}
