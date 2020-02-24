package config

import (
	"bytes"
	"testing"

	"github.com/bluemedora/bplogagent/plugin"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPluginConfig(t *testing.T) {
	rawConfig := []byte(`
plugins:
- type: generate
  interval: 1
  record:
    test: asdf
`)

	expectedConfig := Config{
		Plugins: []plugin.PluginConfig{
			&plugin.GenerateConfig{
				Interval: 1,
				Record:   map[string]interface{}{"test": "asdf"},
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
