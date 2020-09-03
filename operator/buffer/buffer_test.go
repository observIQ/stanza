package buffer

import (
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestBufferUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Config
	}{
		{
			"SimpleMemory",
			[]byte("type: memory\nmax_entries: 30\n"),
			Config{
				Type: "memory",
				BufferBuilder: &MemoryBufferConfig{
					MaxEntries: 30,
				},
			},
		},
		{
			"SimpleDisk",
			[]byte("type: disk\nmax_size: 1234\npath: /var/log/testpath\n"),
			Config{
				Type: "disk",
				BufferBuilder: &DiskBufferConfig{
					MaxSize: 1234,
					Path:    "/var/log/testpath",
					Sync:    true,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Config
			err := yaml.Unmarshal(tc.input, &b)
			require.NoError(t, err)
			require.Equal(t, tc.expected, b)
		})
	}
}
