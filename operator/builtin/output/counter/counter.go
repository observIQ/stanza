package counter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

type CounterOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	Path                string          `json:"path" yaml:"path"`
	Duration            helper.Duration `yaml:"duration,omitempty"`
}

var defaultCounterDuration = helper.NewDuration(1 * time.Minute)

func init() {
	operator.Register("counter_output", func() operator.Builder { return NewCounterOutputConfig("") })
}

func NewCounterOutputConfig(operatorID string) *CounterOutputConfig {
	return &CounterOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "counter_output"),
		Duration:     defaultCounterDuration,
	}
}

func (c CounterOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	counterOperator := &CounterOperator{
		OutputOperator: outputOperator,
		ctx:            ctx,
		cancel:         cancel,
		interval:       c.Duration.Duration,
		path:           c.Path,
		numEntries:     big.NewInt(0),
		wg:             sync.WaitGroup{},
	}

	return []operator.Operator{counterOperator}, nil
}

type CounterOperator struct {
	helper.OutputOperator
	ctx      context.Context
	interval time.Duration
	start    time.Time
	encoder  *json.Encoder
	path     string
	wg       sync.WaitGroup
	cancel   context.CancelFunc

	numEntries *big.Int
}

func (co *CounterOperator) Process(_ context.Context, _ *entry.Entry) error {
	co.numEntries = co.numEntries.Add(co.numEntries, big.NewInt(1))
	return nil
}

func (co *CounterOperator) Start() error {
	err := co.determineOutput()
	if err != nil {
		return err
	}

	co.start = time.Now()
	co.wg.Add(1)
	go co.startCounting()

	return nil
}

// Stop tells the ForwardOutput to stop gracefully
func (co *CounterOperator) Stop() error {
	co.cancel()
	co.wg.Wait()
	return nil
}

func (co *CounterOperator) startCounting() {
	defer co.wg.Done()

	ticker := time.NewTicker(co.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := co.logCount()
			if err != nil {
				return
			}
		case <-co.ctx.Done():
			return
		}
	}
}

func (co *CounterOperator) logCount() error {
	now := time.Now()
	elapsedMinutes := math.Floor(now.Sub(co.start).Minutes())
	entriesPerMinute := float64(co.numEntries.Int64()) / elapsedMinutes
	msg := map[string]interface{}{
		"entries":        co.numEntries,
		"elapsedMinutes": elapsedMinutes,
		"entries/minute": entriesPerMinute,
	}
	return co.encoder.Encode(msg)
}

func (co *CounterOperator) determineOutput() error {
	if co.path == "" {
		co.encoder = json.NewEncoder(os.Stdout)
		return nil
	}

	file, err := os.OpenFile(co.path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("unable to write counter info to file located at %s: %w", co.path, err)
	}
	co.encoder = json.NewEncoder(file)
	return nil
}
