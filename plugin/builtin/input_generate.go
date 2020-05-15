package builtin

import (
	"context"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("generate_input", &GenerateInputConfig{})
}

// GenerateInputConfig is the configuration of a generate input plugin.
type GenerateInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`

	Record map[string]interface{} `mapstructure:"record" json:"record"          yaml:"record"`
	Count  int                    `mapstructure:"count"  json:"count,omitempty" yaml:"count,omitempty"`
}

// Build will build a generate input plugin.
func (c GenerateInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	generateInput := &GenerateInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		record:      c.Record,
		count:       c.Count,
	}
	return generateInput, nil
}

// GenerateInput is a plugin that generates log entries.
type GenerateInput struct {
	helper.BasicPlugin
	helper.BasicInput
	count  int
	record map[string]interface{}
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

// Start will start generating log entries.
func (g *GenerateInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.wg = &sync.WaitGroup{}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			entry := &entry.Entry{
				Timestamp: time.Now(),
				Record:    copyMap(g.record),
			}

			err := g.Output.Process(entry)
			if err != nil {
				g.Warnw("process entry", "error", err)
			}

			i++
			if i == g.count {
				return
			}

		}
	}()

	return nil
}

// Stop will stop generating logs.
func (g *GenerateInput) Stop() error {
	g.cancel()
	g.wg.Wait()
	return nil
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
