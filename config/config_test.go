package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
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
          from: $.message.nested
          to: message.nested3
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
					WriteTo:  entry.Field(entry.NewField("message")),
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
							Field: entry.NewField("message", "nested"),
							Value: "testvalue",
						},
					},
					{
						OpApplier: &builtin.OpAdd{
							Field: entry.NewField("message", "nested2"),
							Value: "testvalue2",
						},
					},
					{
						OpApplier: &builtin.OpRemove{
							Field: entry.NewField("message", "nested2"),
						},
					},
					{
						OpApplier: &builtin.OpMove{
							From: entry.NewField("message", "nested"),
							To:   entry.NewField("message", "nested3"),
						},
					},
					{
						OpApplier: &builtin.OpRetain{
							Fields: []entry.Field{entry.NewField("message", "nested3")},
						},
					},
					{
						OpApplier: &builtin.OpFlatten{
							Field: entry.NewField("message"),
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

func TestReadConfigsFromGlobs(t *testing.T) {

	output1 := []byte(`
plugins:
  - id: output1
    type: logger_output
`)

	file1 := []byte(`
database_file: test1.db
plugins:
  - id: fileinput1
    type: file_input
    output: output1
`)

	file2 := []byte(`
database_file: test2.db
plugins:
  - id: fileinput2
    type: file_input
    output: output1
`)

	cases := []struct {
		name                 string
		globs                []string
		expectedPluginIDs    []string
		expectedDatabaseFile string
		expectedError        require.ErrorAssertionFunc
	}{
		{
			"multiple inputs",
			[]string{"file1", "file2", "output1"},
			[]string{"fileinput1", "fileinput2", "output1"},
			"test2.db",
			require.NoError,
		},
		{
			"single input",
			[]string{"file1", "output1"},
			[]string{"fileinput1", "output1"},
			"test1.db",
			require.NoError,
		},
		{
			"globbed inputs",
			[]string{"file*", "output1"},
			[]string{"fileinput1", "fileinput2", "output1"},
			"test2.db", // because glob returns in lexicographical order
			require.NoError,
		},
		{
			"globbed all",
			[]string{"*"},
			[]string{"fileinput1", "fileinput2", "output1"},
			"test2.db", // because glob returns in lexicographical order
			require.NoError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temp dir
			dir, err := ioutil.TempDir("", "")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			// Write the config files to the temp dir
			ioutil.WriteFile(filepath.Join(dir, "output1"), output1, 0666)
			ioutil.WriteFile(filepath.Join(dir, "file1"), file1, 0666)
			ioutil.WriteFile(filepath.Join(dir, "file2"), file2, 0666)

			// Prefix the globs with the temp dir
			globs := make([]string, len(tc.globs))
			for i, glob := range tc.globs {
				globs[i] = filepath.Join(dir, glob)
			}
			cfg, err := ReadConfigsFromGlobs(globs)
			tc.expectedError(t, err)

			// Pull out the plugin IDs from the unmarshaled plugins
			pluginIDs := make([]string, len(cfg.Plugins))
			for i, plugin := range cfg.Plugins {
				pluginIDs[i] = plugin.ID()
			}

			require.Equal(t, tc.expectedDatabaseFile, cfg.DatabaseFile)
			require.ElementsMatch(t, tc.expectedPluginIDs, pluginIDs)
		})
	}

}
