package builtin

import (
	"context"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("generate_input", &GenerateInputConfig{})
}

// GenerateInputConfig is the configuration of a generate input plugin.
type GenerateInputConfig struct {
	base.InputConfig `mapstructure:",squash" yaml:",inline"`
	Record           map[string]interface{}
	Count            int `yaml:",omitempty"`
}

// Build will build a generate input plugin.
func (c GenerateInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	generateInput := &GenerateInput{
		InputPlugin: inputPlugin,
		record:      c.Record,
		count:       c.Count,
	}
	return generateInput, nil
}

// GenerateInput is a plugin that generates log entries.
type GenerateInput struct {
	base.InputPlugin
	count  int
	record map[string]interface{}
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

// Start will start generating log entries.
func (g *GenerateInput) Start() error {
	// TODO protect against multiple starts?
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

			err := g.Output.Consume(entry)
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
