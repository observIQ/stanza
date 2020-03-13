package config

import (
	"bytes"
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/builtin"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPluginConfig(t *testing.T) {
	rawConfig := []byte(`
plugins:
- id: mygenerate
  type: generate
  count: 1
  output: next
  record:
    test: asdf
`)

	expectedConfig := Config{
		Plugins: []plugin.PluginConfig{
			&builtin.GenerateConfig{
				DefaultPluginConfig: plugin.DefaultPluginConfig{
					PluginID:   "mygenerate",
					PluginType: "generate",
				},
				DefaultOutputterConfig: plugin.DefaultOutputterConfig{
					Output: "next",
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
	err := v.UnmarshalExact(&config, plugin.UnmarshalHook)

	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}
