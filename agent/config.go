package agent

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/bluemedora/bplogagent/pipeline"
	yaml "gopkg.in/yaml.v2"
)

// Config is the configuration of a log agent.
type Config struct {
	Pipeline     pipeline.Config `json:"pipeline"                yaml:"pipeline"`
	DatabaseFile string          `json:"database_file,omitempty" yaml:"database_file,omitempty"`
}

// NewConfigFromFile will create a new agent config from a YAML file.
func NewConfigFromFile(file string) (*Config, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(contents, config); err != nil {
		return nil, fmt.Errorf("failed to read config file as yaml: %s", err)
	}

	return config, nil
}

// NewConfigFromGlobs will create an agent config from multiple files matching a pattern.
func NewConfigFromGlobs(globs []string) (*Config, error) {
	paths := make([]string, 0, len(globs))
	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("No config files found")
	}

	config := &Config{}
	for _, path := range paths {
		newConfig, err := NewConfigFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %s", path, err)
		}

		config = mergeConfigs(config, newConfig)
	}

	return config, nil
}

// mergeConfigs will merge two agent configs.
func mergeConfigs(dst *Config, src *Config) *Config {
	if src.DatabaseFile != "" {
		dst.DatabaseFile = src.DatabaseFile
	}

	dst.Pipeline = append(dst.Pipeline, src.Pipeline...)
	return dst
}
