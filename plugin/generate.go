package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

func init() {
	RegisterConfig("generate", &GenerateConfig{})
}

type GenerateConfig struct {
	DefaultSourceConfig `mapstructure:",squash"`
	Record              map[string]interface{}
	Count               int
}

func (c GenerateConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	defaultSource, err := c.DefaultSourceConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default source: %s", err)
	}

	plugin := &GeneratePlugin{
		config:        c,
		SugaredLogger: logger.With("plugin_type", "generate", "plugin_id", c.ID()),
		DefaultSource: defaultSource,
	}
	return plugin, nil
}

type GeneratePlugin struct {
	DefaultSource
	config GenerateConfig

	cancel context.CancelFunc
	*zap.SugaredLogger
}

func (p *GeneratePlugin) Start(wg *sync.WaitGroup) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer wg.Done()

		i := 0
		for {
			entry := entry.Entry{
				Timestamp: time.Now(),
				Record:    copyMap(p.config.Record),
			}

			select {
			case <-ctx.Done():
				return
			case p.output <- entry:
			}

			i += 1
			if i == p.config.Count {
				return
			}

		}
	}()

	return nil
}

func (p *GeneratePlugin) Stop() {
	// TODO should this block until exit?
	p.cancel()
}

// TODO This is a really dumb implementation right now.
// Should this do something different with pointers or arrays?
func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
