package agent

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/operator"
	_ "github.com/observiq/stanza/operator/builtin/transformer/noop"
	"github.com/observiq/stanza/pipeline"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromFile(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	configFile := filepath.Join(tempDir, "config.yaml")
	configContents := `
pipeline:
  - type: noop
`
	err := ioutil.WriteFile(configFile, []byte(configContents), 0755)
	require.NoError(t, err)

	config, err := NewConfigFromFile(configFile)
	require.NoError(t, err)
	require.Equal(t, len(config.Pipeline), 1)
}

func TestNewConfigWithMissingFile(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	configFile := filepath.Join(tempDir, "config.yaml")

	_, err := NewConfigFromFile(configFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find config file")
}

func TestNewConfigWithInvalidYAML(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	configFile := filepath.Join(tempDir, "config.yaml")
	configContents := `
pipeline:
  invalid: structure
`
	err := ioutil.WriteFile(configFile, []byte(configContents), 0755)
	require.NoError(t, err)

	_, err = NewConfigFromFile(configFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read config file as yaml")
}

func TestNewConfigFromGlobs(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	configFile := filepath.Join(tempDir, "config.yaml")
	configContents := `
pipeline:
  - type: noop
`
	err := ioutil.WriteFile(configFile, []byte(configContents), 0755)
	require.NoError(t, err)

	globs := []string{filepath.Join(tempDir, "*.yaml")}
	config, err := NewConfigFromGlobs(globs)
	require.NoError(t, err)
	require.Equal(t, len(config.Pipeline), 1)
}

func TestNewConfigFromGlobsWithInvalidGlob(t *testing.T) {
	globs := []string{"[]"}
	_, err := NewConfigFromGlobs(globs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "syntax error in pattern")
}

func TestNewConfigFromGlobsWithNoMatches(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	globs := []string{filepath.Join(tempDir, "*.yaml")}
	_, err := NewConfigFromGlobs(globs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No config files found")
}

func TestNewConfigFromGlobsWithInvalidConfig(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	configFile := filepath.Join(tempDir, "config.yaml")
	configContents := `
pipeline:
  invalid: structure
`
	err := ioutil.WriteFile(configFile, []byte(configContents), 0755)
	require.NoError(t, err)

	globs := []string{filepath.Join(tempDir, "*.yaml")}
	_, err = NewConfigFromGlobs(globs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read config file as yaml")
}

func TestMergeConfigs(t *testing.T) {
	config1 := Config{
		Pipeline: pipeline.Config{
			operator.Config{},
		},
	}

	config2 := Config{
		Pipeline: pipeline.Config{
			operator.Config{},
		},
	}

	config3 := mergeConfigs(&config1, &config2)
	require.Equal(t, len(config3.Pipeline), 2)
}
