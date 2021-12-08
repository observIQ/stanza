package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func graphTest(config, output string) func(t *testing.T) {
	return func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yaml")
		err = ioutil.WriteFile(configPath, []byte(config), 0666)
		require.NoError(t, err)

		rootFlags := &RootFlags{
			ConfigFiles: []string{configPath},
		}
		graphCmd := NewGraphCommand(rootFlags)

		// replace stdout
		buf := bytes.NewBuffer([]byte{})
		stdout = buf

		err = graphCmd.Execute()
		require.NoError(t, err)

		require.Equal(t, testutil.Trim(output), testutil.Trim(buf.String()))
	}
}

func TestGraphSimple(t *testing.T) {
	config := `
pipeline:
  - id: generate
    type: generate_input
    output: json_parser
    entry:
      record:
        test: value

  - id: json_parser
    type: json_parser
    output: stdout

  - id: stdout
    type: stdout
`

	expected := `
    strict digraph G {
      // Node definitions.
      "$.json_parser";
      "$.generate";
      "$.stdout";

      // Edge definitions.
      "$.json_parser" -> "$.stdout";
      "$.generate" -> "$.json_parser";
    }`

	graphTest(config, expected)(t)
}

func TestGraphNoIDs(t *testing.T) {
	config := `
pipeline:
  - type: generate_input
    output: json_parser
    entry:
      record:
        test: value

  - type: json_parser
    output: stdout

  - type: stdout
`

	expected := `
    strict digraph G {
      // Node definitions.
      "$.json_parser";
      "$.stdout";
      "$.generate_input";

      // Edge definitions.
      "$.json_parser" -> "$.stdout";
      "$.generate_input" -> "$.json_parser";
    }`

	graphTest(config, expected)(t)
}

func TestGraphNoOutputs(t *testing.T) {
	config := `
pipeline:
  - id: generate
    type: generate_input
    entry:
      record:
        test: value

  - id: json_parser
    type: json_parser

  - id: stdout
    type: stdout
`

	expected := `
    strict digraph G {
      // Node definitions.
      "$.json_parser";
      "$.generate";
      "$.stdout";

      // Edge definitions.
      "$.json_parser" -> "$.stdout";
      "$.generate" -> "$.json_parser";
    }`

	graphTest(config, expected)(t)
}

func TestGraphNoOutputsNoIDs(t *testing.T) {
	config := `
pipeline:
  - type: generate_input
    entry:
      record:
        test: value

  - type: json_parser

  - type: stdout
`

	expected := `
    strict digraph G {
      // Node definitions.
      "$.json_parser";
      "$.stdout";
      "$.generate_input";

      // Edge definitions.
      "$.json_parser" -> "$.stdout";
      "$.generate_input" -> "$.json_parser";
    }`

	graphTest(config, expected)(t)
}

func TestGraphMixed(t *testing.T) {
	config := `
pipeline:
  - type: generate_input
    entry:
      record:
        test: value

  - type: json_parser
    output: my_stdout

  - type: drop_output

  - id: my_stdout
    type: stdout
`

	expected := `
    strict digraph G {
      // Node definitions.
      "$.json_parser";
      "$.drop_output";
      "$.my_stdout";
      "$.generate_input";

      // Edge definitions.
      "$.json_parser" -> "$.my_stdout";
      "$.generate_input" -> "$.json_parser";
    }`

	graphTest(config, expected)(t)
}
