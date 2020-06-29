package input

import (
	"context"
	"fmt"
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
	helper.InputConfig `yaml:",inline"`
	Entry              entry.Entry `json:"entry"           yaml:"entry"`
	Count              int         `json:"count,omitempty" yaml:"count,omitempty"`
	Static             bool        `json:"static" yaml:"static"`
}

// Build will build a generate input plugin.
func (c *GenerateInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	c.Entry.Record = recursiveMapInterfaceToMapString(c.Entry.Record)

	generateInput := &GenerateInput{
		InputPlugin: inputPlugin,
		entry:       c.Entry,
		count:       c.Count,
		static:      c.Static,
	}
	return generateInput, nil
}

// GenerateInput is a plugin that generates log entries.
type GenerateInput struct {
	helper.InputPlugin
	entry  entry.Entry
	count  int
	static bool
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

			entry := g.entry.Copy()
			if !g.static {
				entry.Timestamp = time.Now()
			}
			g.Write(ctx, entry)

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

func recursiveMapInterfaceToMapString(m interface{}) interface{} {
	switch m := m.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, v := range m {
			newMap[k] = recursiveMapInterfaceToMapString(v)
		}
		return newMap
	case map[interface{}]interface{}:
		newMap := make(map[string]interface{})
		for k, v := range m {
			kStr, ok := k.(string)
			if !ok {
				kStr = fmt.Sprintf("%v", k)
			}
			newMap[kStr] = recursiveMapInterfaceToMapString(v)
		}
		return newMap
	default:
		return m
	}
}
