package config

import (
	"bytes"
	"testing"

	"github.com/bluemedora/log-agent/plugin"
	"github.com/mitchellh/mapstructure"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPluginConfig(t *testing.T) {
	rawConfig := []byte(`
plugins:
- type: generate
  rate: 1000
  message: test
`)

	expectedConfig := Config{
		Plugins: []plugin.PluginConfig{
			&plugin.GenerateSourceConfig{
				Rate:    1000,
				Message: "test",
			},
		},
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewReader(rawConfig))

	var config Config
	err := v.Unmarshal(&config, func(c *mapstructure.DecoderConfig) {
		c.DecodeHook = plugin.PluginConfigToStructHookFunc()
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}
