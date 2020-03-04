package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
)

func init() {
	RegisterConfig("generate", &GenerateConfig{})
}

type GenerateConfig struct {
	DefaultPluginConfig    `mapstructure:",squash"`
	DefaultOutputterConfig `mapstructure:",squash"`
	Record                 map[string]interface{}
	Count                  int
}

func (c GenerateConfig) Build(context BuildContext) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to build default outputter: %s", err)
	}

	plugin := &GeneratePlugin{
		config:           c,
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
	}
	return plugin, nil
}

type GeneratePlugin struct {
	DefaultPlugin
	DefaultOutputter
	config GenerateConfig

	cancel context.CancelFunc
}

func (p *GeneratePlugin) Start(wg *sync.WaitGroup) error {
	// TODO protect against multiple starts?
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
			case p.Output() <- entry:
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
