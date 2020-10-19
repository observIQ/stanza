package otlp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalJSON(t *testing.T) {

	cfgBytes := []byte(`{ "headers": { "testKey": "testValue" }, "endpoint": "localhost:1234", "compression": "gzip", "wait_for_ready": true }`)

	cfg := HTTPClientConfig{}
	err := json.Unmarshal(cfgBytes, &cfg)

	require.NoError(t, err)

	expected := HTTPClientConfig{
		confighttp.HTTPClientSettings{
			Headers:  map[string]string{"testKey": "testValue"},
			Endpoint: "localhost:1234",
		},
	}

	require.Equal(t, expected, cfg)
}

func TestUnmarshalYAML(t *testing.T) {

	cfgBytes := []byte(`
headers: 
  testKey: testValue
endpoint: localhost:1234
compression: gzip
wait_for_ready: true`)

	cfg := HTTPClientConfig{}
	err := yaml.Unmarshal(cfgBytes, &cfg)

	require.NoError(t, err)

	expected := HTTPClientConfig{
		confighttp.HTTPClientSettings{
			Headers:  map[string]string{"testKey": "testValue"},
			Endpoint: "localhost:1234",
		},
	}

	require.Equal(t, expected, cfg)
}
