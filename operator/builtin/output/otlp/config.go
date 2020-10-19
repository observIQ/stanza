package otlp

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/config/confighttp"
)

// HTTPClientConfig contains all
type HTTPClientConfig struct {
	confighttp.HTTPClientSettings
}

// NewHTTPClientConfig
func NewHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{confighttp.HTTPClientSettings{}}
}

// UnmarshalJSON will unmarshal json into a HTTPClientConfig struct
func (g *HTTPClientConfig) UnmarshalJSON(data []byte) error {
	any := make(map[string]interface{})
	if err := json.Unmarshal(data, &any); err != nil {
		return err
	}

	settings := confighttp.HTTPClientSettings{
		Endpoint: "http://localhost:55681/v1/logs",
	}
	if err := mapstructure.Decode(any, &settings); err != nil {
		return err
	}
	g.HTTPClientSettings = settings
	return nil
}

// UnmarshalYAML will unmarshal json into a HTTPClientConfig struct
func (g *HTTPClientConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var any interface{}
	if err := unmarshal(&any); err != nil {
		return err
	}

	settings := confighttp.HTTPClientSettings{
		Endpoint: "http://localhost:55681/v1/logs",
	}
	if err := mapstructure.Decode(any, &settings); err != nil {
		return err
	}
	g.HTTPClientSettings = settings
	return nil
}
