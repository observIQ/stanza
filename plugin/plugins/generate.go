package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("generate", &GenerateConfig{})
}

type GenerateConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
	Record                    map[string]interface{}
	Count                     int `yaml:",omitempty"`
}

func (c GenerateConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	plugin := &GenerateSource{
		config:           c,
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
	}
	return plugin, nil
}

type GenerateSource struct {
	pg.DefaultPlugin
	pg.DefaultOutputter
	config GenerateConfig

	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func (p *GenerateSource) Start() error {
	// TODO protect against multiple starts?
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.wg = &sync.WaitGroup{}
	p.wg.Add(1)

	go func() {
		p.wg.Done()
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			entry := &entry.Entry{
				Timestamp: time.Now(),
				Record:    copyMap(p.config.Record),
			}

			err := p.Output(entry)
			if err != nil {
				p.Warnw("process entry", "error", err)
			}

			i += 1
			if i == p.config.Count {
				return
			}

		}
	}()

	return nil
}

func (p *GenerateSource) Stop() {
	p.cancel()
	p.wg.Wait()
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
