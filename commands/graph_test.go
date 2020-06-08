package commands

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGraph(t *testing.T) {
	config := []byte(`
pipeline:
  - id: generate
    type: generate_input
    output: json_parser
    record:
      test: value

  - id: json_parser
    type: json_parser
    output: google_cloud

  - id: google_cloud
    project_id: testproject
    type: google_cloud_output
`)
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")
	err = ioutil.WriteFile(configPath, config, 0666)
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

	// Output:
	expected := `strict digraph G {
 // Node definitions.
 "$.json_parser";
 "$.generate";
 "$.google_cloud";

 // Edge definitions.
 "$.json_parser" -> "$.google_cloud";
 "$.generate" -> "$.json_parser";
}`

	require.Equal(t, expected, buf.String())
}
