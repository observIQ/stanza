package otlp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	yaml "gopkg.in/yaml.v2"
)

func TestNew(t *testing.T) {
	expected := HTTPClientConfig{
		confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:55681/v1/logs",
		},
	}

	require.Equal(t, expected, NewHTTPClientConfig())
}

func TestEmptyEndpoint(t *testing.T) {
	cfg := NewHTTPClientConfig()
	cfg.Endpoint = ""
	require.Error(t, cfg.cleanEndpoint())
}

func TestPrependHTTP(t *testing.T) {
	cfg := NewHTTPClientConfig()
	cfg.Endpoint = "localhost:1234/v1/logs"
	cfg.TLSSetting.Insecure = true
	require.NoError(t, cfg.cleanEndpoint())
	require.Equal(t, "http://localhost:1234/v1/logs", cfg.Endpoint)
}

func TestPrependHTTPS(t *testing.T) {
	cfg := NewHTTPClientConfig()
	cfg.Endpoint = "localhost:1234/v1/logs"
	require.NoError(t, cfg.cleanEndpoint())
	require.Equal(t, "https://localhost:1234/v1/logs", cfg.Endpoint)
}

func TestAppendService(t *testing.T) {
	cfg := NewHTTPClientConfig()
	cfg.Endpoint = "localhost:1234"
	require.NoError(t, cfg.cleanEndpoint())
	require.Equal(t, "https://localhost:1234/v1/logs", cfg.Endpoint)
}

func TestUnmarshalJSON(t *testing.T) {

	cfgBytes := []byte(`{ "headers": { "testKey": "testValue" }, "endpoint": "localhost:1234" }`)

	cfg := HTTPClientConfig{}
	require.NoError(t, json.Unmarshal(cfgBytes, &cfg))
	require.NoError(t, cfg.cleanEndpoint())

	expected := HTTPClientConfig{
		confighttp.HTTPClientSettings{
			Headers:  map[string]string{"testKey": "testValue"},
			Endpoint: "https://localhost:1234/v1/logs",
		},
	}

	require.Equal(t, expected, cfg)
}

func TestUnmarshalYAML(t *testing.T) {

	cfgBytes := []byte(`
headers: 
  testKey: testValue
endpoint: localhost:1234`)

	cfg := HTTPClientConfig{}
	require.NoError(t, yaml.Unmarshal(cfgBytes, &cfg))
	require.NoError(t, cfg.cleanEndpoint())

	expected := HTTPClientConfig{
		confighttp.HTTPClientSettings{
			Headers:  map[string]string{"testKey": "testValue"},
			Endpoint: "https://localhost:1234/v1/logs",
		},
	}

	require.Equal(t, expected, cfg)
}
