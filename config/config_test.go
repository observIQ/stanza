package config

import (
	"bytes"
	"testing"

	"go.bluemedora.com/bplogagent/plugin"

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
	err := v.UnmarshalExact(&config, UnmarshalHook)

	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}
