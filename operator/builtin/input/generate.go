package input

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("generate_input", func() operator.Builder { return NewGenerateInputConfig("") })
}

func NewGenerateInputConfig(operatorID string) *GenerateInputConfig {
	return &GenerateInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "generate_input"),
	}
}

// GenerateInputConfig is the configuration of a generate input operator.
type GenerateInputConfig struct {
	helper.InputConfig `yaml:",inline"`
	Entry              entry.Entry `json:"entry"           yaml:"entry"`
	Count              int         `json:"count,omitempty" yaml:"count,omitempty"`
	Static             bool        `json:"static"          yaml:"static,omitempty"`
}

// Build will build a generate input operator.
func (c *GenerateInputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	c.Entry.Record = recursiveMapInterfaceToMapString(c.Entry.Record)
	c.Entry.AddLabel("log_type", inputOperator.LogType)

	generateInput := &GenerateInput{
		InputOperator: inputOperator,
		entry:         c.Entry,
		count:         c.Count,
		static:        c.Static,
	}
	return generateInput, nil
}

// GenerateInput is an operator that generates log entries.
type GenerateInput struct {
	helper.InputOperator
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
