package bplogagent

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
  rate: 1000
  message:
    test: asdf
`)

	expectedConfig := Config{
		Plugins: []plugin.PluginConfig{
			&plugin.GenerateConfig{
				Rate:    1000,
				Message: map[string]interface{}{"test": "asdf"},
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
