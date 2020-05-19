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
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

type bufferHandler struct {
	flushed []*entry.Entry
	mux     sync.Mutex
	notify  chan struct{}
}

func (b *bufferHandler) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {
	b.mux.Lock()
	b.flushed = append(b.flushed, entries...)
	b.mux.Unlock()
	b.notify <- struct{}{}
	return nil
}

func (b *bufferHandler) Logger() *zap.SugaredLogger {
	return nil

}
func TestBuffer(t *testing.T) {
	config := &BufferConfig{}
	config.setDefaults()
	config.DelayThreshold = plugin.Duration{
		Duration: 100 * time.Millisecond,
	}

	buf := NewMemoryBuffer(config)
	numEntries := 10000

	bh := bufferHandler{
		flushed: make([]*entry.Entry, 0),
		notify:  make(chan struct{}),
	}
	buf.SetHandler(&bh)

	for i := 0; i < numEntries; i++ {
		err := buf.AddWait(context.Background(), entry.New(), 0)
		require.NoError(t, err)
	}

	for {
		select {
		case <-bh.notify:
			bh.mux.Lock()
			if len(bh.flushed) == numEntries {
				bh.mux.Unlock()
				return
			}
			bh.mux.Unlock()
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
