package buffer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestBuffer(t *testing.T) {
	config := &BufferConfig{}
	config.setDefaults()
	config.DelayThreshold = plugin.Duration{
		Duration: 100 * time.Millisecond,
	}

	buf := NewMemoryBuffer(config)
	numEntries := 10000

	flushed := make([]*entry.Entry, 0, numEntries)
	flushedMux := sync.Mutex{}
	notify := make(chan struct{})
	buf.SetHandler(func(ctx context.Context, entries []*entry.Entry) error {
		flushedMux.Lock()
		flushed = append(flushed, entries...)
		flushedMux.Unlock()
		notify <- struct{}{}
		return nil
	})

	for i := 0; i < numEntries; i++ {
		err := buf.AddWait(context.Background(), entry.New(), 0)
		require.NoError(t, err)
	}

	for {
		select {
		case <-notify:
			flushedMux.Lock()
			if len(flushed) == numEntries {
				flushedMux.Unlock()
				return
			}
			flushedMux.Unlock()
		case <-time.After(time.Second):
			require.FailNow(t, "timed out waiting for all entries to be flushed")
		}
	}
}

func TestBufferSerializationRoundtrip(t *testing.T) {
	cases := []struct {
		name   string
		config BufferConfig
	}{
		{
			"zeros",
			BufferConfig{},
		},
		{
			"defaults",
			func() BufferConfig {
				config := BufferConfig{}
				config.setDefaults()
				return config
			}(),
		},
	}

	for _, tc := range cases {
		t.Run("yaml "+tc.name, func(t *testing.T) {
			cfgBytes, err := yaml.Marshal(tc.config)
			require.NoError(t, err)

			var cfg BufferConfig
			err = yaml.Unmarshal(cfgBytes, &cfg)
			require.NoError(t, err)

			require.Equal(t, tc.config, cfg)
		})

		t.Run("json "+tc.name, func(t *testing.T) {
			cfgBytes, err := json.Marshal(tc.config)
			require.NoError(t, err)

			var cfg BufferConfig
			err = json.Unmarshal(cfgBytes, &cfg)
			require.NoError(t, err)

			require.Equal(t, tc.config, cfg)
		})
	}
}
