package otlp

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/config/configgrpc"
)

// GRPCClientConfig contains all
type GRPCClientConfig struct {
	configgrpc.GRPCClientSettings
}

// NewGRPCClientConfig
func NewGRPCClientConfig() GRPCClientConfig {
	return GRPCClientConfig{configgrpc.GRPCClientSettings{}}
}

// UnmarshalJSON will unmarshal json into a GRPCClientConfig struct
func (g *GRPCClientConfig) UnmarshalJSON(data []byte) error {
	any := make(map[string]interface{})
	if err := json.Unmarshal(data, &any); err != nil {
		return err
	}

	settings := configgrpc.GRPCClientSettings{}
	if err := mapstructure.Decode(any, &settings); err != nil {
		return err
	}
	g.GRPCClientSettings = settings
	return nil
}

// UnmarshalYAML will unmarshal json into a GRPCClientConfig struct
func (g *GRPCClientConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var any interface{}
	if err := unmarshal(&any); err != nil {
		return err
	}

	settings := configgrpc.GRPCClientSettings{}
	if err := mapstructure.Decode(any, &settings); err != nil {
		return err
	}
	g.GRPCClientSettings = settings
	return nil
}
