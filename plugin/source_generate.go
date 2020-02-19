package plugin

import (
	"github.com/bluemedora/log-agent/config"
)

func init() {
	config.RegisterConfig("generate", &GenerateSourceConfig{})
}

type GenerateSourceConfig struct {
	Output  string
	Message string
	Rate    float64
}
