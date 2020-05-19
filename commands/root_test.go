package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRoot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	input := []byte(`log1
log2
log3`)
	err = ioutil.WriteFile(filepath.Join(tempDir, "input.log"), input, 0666)
	require.NoError(t, err)

	config := `
database_file: "%s"
plugins:
  - id: file_input
    type: file_input
    include: ["%s"]
    write_to: message
    output: file_output

  - id: file_output
    type: file_output
    path: "%s"
`
	config = fmt.Sprintf(
		config,
		filepath.Join(tempDir, "logagent.db"),
		filepath.Join(tempDir, "input.log"),
		filepath.Join(tempDir, "output.json"),
	)
	err = ioutil.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(config), 0666)
	require.NoError(t, err)

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"-c", filepath.Join(tempDir, "config.yaml")})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rootCmd.ExecuteContext(ctx)

	expectedPattern := `{"timestamp":".*","record":{"message":"log1"}}
{"timestamp":".*","record":{"message":"log2"}}
{"timestamp":".*","record":{"message":"log3"}}
`

	actual, err := ioutil.ReadFile(filepath.Join(tempDir, "output.json"))
	require.NoError(t, err)

	require.Regexp(t, expectedPattern, string(actual))
}
