package service

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/pipeline"
	"github.com/open-telemetry/opentelemetry-log-collection/plugin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Pipeline pipeline.Config `yaml:"pipeline"`
	Logging  *LoggingConfig  `yaml:"logging,omitempty"`
}

// LoadConfig loads the plugins from the given directory and the config from the given file path.
// This function should not be called twice
func LoadConfig(pluginDir, file string) (*Config, error) {
	if pluginDir != "" {
		// Plugins MUST be loaded before calling LoadConfig, otherwise stanza will fail to recognize plugin
		// types, and fail to load any config using plugins
		if errs := plugin.RegisterPlugins(pluginDir, operator.DefaultRegistry); len(errs) != 0 {
			log.Fatalf("Got errors parsing plugins %s", errs)
		}
	}

	contents, err := ioutil.ReadFile(file) // #nosec - configs load based on user specified directory
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// initialize logging defaults here, so unmarshalling "overrides" them
	config := Config{
		Logging: DefaultLoggingConfig(),
	}

	if err := yaml.UnmarshalStrict(contents, &config); err != nil {
		return nil, fmt.Errorf("failed to marshal config as yaml: %w", err)
	}

	return &config, nil
}
