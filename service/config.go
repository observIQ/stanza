package service

import (
	"fmt"
	"io/ioutil"

	"github.com/open-telemetry/opentelemetry-log-collection/pipeline"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Pipeline pipeline.Config `yaml:"pipeline"`
	Logging  *LoggingConfig  `yaml:"logging,omitempty"`
}

func LoadConfig(file string) (*Config, error) {
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
