package config

import (
	"bytes"
	"testing"

	"github.com/bluemedora/log-agent/plugin"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPluginConfig(t *testing.T) {
	a := []byte(`
plugins:
	- type: generator
		rate: 100
		message: test
	`)

	expectedConfig := Config{
		Plugins: []PluginConfig{
			&plugin.GenerateSourceConfig{
				Rate:    100,
				Message: "test",
			},
		},
	}
	viper.ReadConfig(bytes.NewReader(a))

	var config Config
	err := viper.UnmarshalExact(&config)
	assert.NoError(t, err)

}
