package config

import (
	"bytes"
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/builtin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPluginConfig(t *testing.T) {
	rawConfig := []byte(`
plugins:
- id: mygenerate
  type: generate_input
  count: 1
  output: next
  record:
    test: asdf
`)

	expectedConfig := Config{
		Plugins: []plugin.Config{
			&builtin.GenerateInputConfig{
				BasicIdentityConfig: helper.BasicIdentityConfig{
					PluginID:   "mygenerate",
					PluginType: "generate_input",
				},
				BasicInputConfig: helper.BasicInputConfig{
					OutputID: "next",
				},
				Count:  1,
				Record: map[string]interface{}{"test": "asdf"},
			},
		},
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewReader(rawConfig))

	var config Config
	err := v.UnmarshalExact(&config, UnmarshalHook)

	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}

func TestConfigIsZero(t *testing.T) {
	config := Config{
		Plugins:    make([]plugin.Config, 0),
		BundlePath: "",
	}

	nestedConfig := struct {
		LogAgentConfig Config `yaml:",omitempty"`
		OtherField     string `yaml:"other_field"`
	}{
		LogAgentConfig: config,
		OtherField:     "test",
	}

	expected := []byte("other_field: test\n")
	marshalled, err := yaml.Marshal(nestedConfig)
	assert.NoError(t, err)

	assert.Equal(t, expected, marshalled)
}
