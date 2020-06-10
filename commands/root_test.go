package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skipping for windows, because it returns an unexplained error.
		// "The service process could not connect to the service controller"
		// This error does not occur when running the binary directly.
		t.Skip("Skipping root test on windows")
	}

	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	input := []byte(`log1
log2
log3`)
	err = ioutil.WriteFile(filepath.Join(tempDir, "input.log"), input, 0666)
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "logagent.db")
	inputPath := filepath.Join(tempDir, "input.log")
	outputPath := filepath.Join(tempDir, "output.json")
	configPath := filepath.Join(tempDir, "config.yaml")

	config := `
database_file: '%s'
pipeline:
  - id: file_input
    type: file_input
    include: ['%s']
    write_to: message
    output: file_output

  - id: file_output
    type: file_output
    path: '%s'
`
	config = fmt.Sprintf(config, dbPath, inputPath, outputPath)
	err = ioutil.WriteFile(configPath, []byte(config), 0666)
	require.NoError(t, err)

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"-c", configPath})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		err = rootCmd.ExecuteContext(ctx)
		require.NoError(t, err)
	}()

	expectedPattern := `{"timestamp":".*","record":{"message":"log1"}}
{"timestamp":".*","record":{"message":"log2"}}
{"timestamp":".*","record":{"message":"log3"}}
`

	time.Sleep(500 * time.Millisecond)

	actual, err := ioutil.ReadFile(outputPath)
	require.NoError(t, err)

	require.Regexp(t, expectedPattern, string(actual))
}
