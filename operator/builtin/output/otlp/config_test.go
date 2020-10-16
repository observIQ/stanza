package otlp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configgrpc"
	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalJSON(t *testing.T) {

	cfgBytes := []byte(`{ "headers": { "testKey": "testValue" }, "endpoint": "localhost:1234", "compression": "gzip", "wait_for_ready": true }`)

	cfg := GRPCClientConfig{}
	err := json.Unmarshal(cfgBytes, &cfg)

	require.NoError(t, err)

	expected := GRPCClientConfig{
		configgrpc.GRPCClientSettings{
			Headers:      map[string]string{"testKey": "testValue"},
			Endpoint:     "localhost:1234",
			Compression:  "gzip",
			WaitForReady: true,
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

	cfg := GRPCClientConfig{}
	err := yaml.Unmarshal(cfgBytes, &cfg)

	require.NoError(t, err)

	expected := GRPCClientConfig{
		configgrpc.GRPCClientSettings{
			Headers:      map[string]string{"testKey": "testValue"},
			Endpoint:     "localhost:1234",
			Compression:  "gzip",
			WaitForReady: true,
		},
	}

	require.Equal(t, expected, cfg)
}
