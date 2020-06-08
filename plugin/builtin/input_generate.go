package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("generate_input", &GenerateInputConfig{})
}

// GenerateInputConfig is the configuration of a generate input plugin.
type GenerateInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Record interface{} `json:"record"          yaml:"record"`
	Count  int         `json:"count,omitempty" yaml:"count,omitempty"`
}

// Build will build a generate input plugin.
func (c *GenerateInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	generateInput := &GenerateInput{
		InputPlugin: inputPlugin,
		record:      recursiveMapInterfaceToMapString(c.Record),
		count:       c.Count,
	}
	return generateInput, nil
}

// GenerateInput is a plugin that generates log entries.
type GenerateInput struct {
	helper.InputPlugin
	count  int
	record interface{}
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

			record := copyRecord(g.record)
			if err := g.Write(ctx, record); err != nil {
				g.Warnw("Failed to process entry", "error", err)
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

func copyRecord(r interface{}) interface{} {
	switch r := r.(type) {
	case map[string]interface{}:
		return copyMap(r)
	case string:
		return r
	case []byte:
		new := make([]byte, 0, len(r))
		copy(new, r)
		return new
	default:
		// fall back to JSON roundtrip
		var i interface{}
		b, err := json.Marshal(r)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(b, &i)
		if err != nil {
			panic(err)
		}
		return i
	}
}

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
