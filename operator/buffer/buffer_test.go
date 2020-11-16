package buffer

import (
	"encoding/json"
	"testing"
  "time"

	"github.com/stretchr/testify/require"
  "github.com/observiq/stanza/operator/helper"
	yaml "gopkg.in/yaml.v2"
)

func TestBufferUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name        string
		yaml        []byte
		json        []byte
		expected    Config
		expectError bool
	}{
		{
			"SimpleMemory",
			[]byte("type: memory\nmax_entries: 30\n"),
			[]byte(`{"type": "memory", "max_entries": 30}`),
			Config{
				Builder: &MemoryBufferConfig{
					Type:       "memory",
					MaxEntries: 30,
          MaxChunkDelay: helper.NewDuration(time.Second),
          MaxChunkSize: 1000,
				},
			},
			false,
		},
		{
			"SimpleDisk",
			[]byte("type: disk\nmax_size: 1234\npath: /var/log/testpath\n"),
			[]byte(`{"type": "disk", "max_size": 1234, "path": "/var/log/testpath"}`),
			Config{
				Builder: &DiskBufferConfig{
					Type:    "disk",
					MaxSize: 1234,
					Path:    "/var/log/testpath",
					Sync:    true,
          MaxChunkDelay: helper.NewDuration(time.Second),
          MaxChunkSize: 1000,
				},
			},
			false,
		},
		{
			"UnknownType",
			[]byte("type: invalid\n"),
			[]byte(`{"type": "invalid"}`),
			Config{
				Builder: &DiskBufferConfig{
					Type:    "disk",
					MaxSize: 1234,
					Path:    "/var/log/testpath",
					Sync:    true,
          MaxChunkDelay: helper.NewDuration(time.Second),
          MaxChunkSize: 1000,
        },
			},
			true,
		},
		{
			"InvalidType",
			[]byte("type: !!float 123\n"),
			[]byte(`{"type": 12}`),
			Config{
				Builder: &DiskBufferConfig{
					Type:    "disk",
					MaxSize: 1234,
					Path:    "/var/log/testpath",
					Sync:    true,
          MaxChunkDelay: helper.NewDuration(time.Second),
          MaxChunkSize: 1000,
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("YAML", func(t *testing.T) {
				var b Config
				err := yaml.Unmarshal(tc.yaml, &b)
				if tc.expectError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.expected, b)
			})

			t.Run("JSON", func(t *testing.T) {
				var b Config
				err := json.Unmarshal(tc.json, &b)
				if tc.expectError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.expected, b)
			})
		})
	}
}

func TestBuffer(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		cfg := NewConfig()
		expected := Config{
			Builder: &MemoryBufferConfig{
				Type:       "memory",
				MaxEntries: 1 << 20,
        MaxChunkDelay: helper.NewDuration(time.Second),
        MaxChunkSize: 1000,
			},
		}
		require.Equal(t, expected, cfg)
	})
}
