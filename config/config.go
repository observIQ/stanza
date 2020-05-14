package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Plugins      []plugin.Config `mapstructure:"plugins"       json:"plugins"                 yaml:"plugins"`
	DatabaseFile string          `mapstructure:"database_file" json:"database_file,omitempty" yaml:"database_file,omitempty"`
}

func NewConfig() *Config {
	return &Config{
		Plugins: make([]plugin.Config, 0, 10),
	}
}

var DecodeHookFunc = plugin.ConfigDecoder

func ReadConfigsFromGlobs(globs []string) (*Config, error) {
	paths := make([]string, 0, len(globs))
	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}

	if len(paths) == 0 {
		return nil, errors.NewError(
			"No config files found",
			"Check that --config points to a valid file",
		)
	}

	cfg := NewConfig()
	for _, path := range paths {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		newCfg := NewConfig()
		err = yaml.Unmarshal(contents, newCfg)
		if err != nil {
			return nil, err
		}

		cfg = mergeConfigs(cfg, newCfg)
	}

	return cfg, nil
}

func mergeConfigs(dst *Config, src *Config) *Config {
	if src.DatabaseFile != "" {
		dst.DatabaseFile = src.DatabaseFile
	}

	dst.Plugins = append(dst.Plugins, src.Plugins...)

	return dst
}
