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

type CountOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	Path                string          `json:"path,omitempty" yaml:"path"`
	Duration            helper.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
}

var defaultCounterDuration = helper.NewDuration(1 * time.Minute)

func init() {
	operator.Register("count_output", func() operator.Builder { return NewCounterOutputConfig("") })
}

func NewCounterOutputConfig(operatorID string) *CountOutputConfig {
	return &CountOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "count_output"),
		Duration:     defaultCounterDuration,
	}
}

func (c CountOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	logChan := make(chan struct{}, 1)
	counterOperator := &CountOutput{
		OutputOperator: outputOperator,
		ctx:            ctx,
		cancel:         cancel,
		interval:       c.Duration.Raw(),
		path:           c.Path,
		numEntries:     big.NewInt(0),
		logChan:        logChan,
		wg:             sync.WaitGroup{},
	}

	return []operator.Operator{counterOperator}, nil
}

type CountOutput struct {
	helper.OutputOperator
	ctx      context.Context
	interval time.Duration
	start    time.Time
	logChan  chan struct{}
	encoder  *json.Encoder
	path     string
	wg       sync.WaitGroup
	cancel   context.CancelFunc

	numEntries *big.Int
}

func (co *CountOutput) Process(_ context.Context, _ *entry.Entry) error {
	co.numEntries = co.numEntries.Add(co.numEntries, big.NewInt(1))
	return nil
}

func (co *CountOutput) Start() error {
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
func (co *CountOutput) Stop() error {
	co.cancel()
	co.wg.Wait()
	return nil
}

func (co *CountOutput) startCounting() {
	defer co.wg.Done()

	ticker := time.NewTicker(co.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-co.logChan:
		case <-co.ctx.Done():
			return
		}

		err := co.logCount()
		if err != nil {
			return
		}
	}
}

type countObject struct {
	Entries          *big.Int `json:"entries"`
	ElapsedMinutes   float64  `json:"elapsedMinutes"`
	EntriesPerMinute float64  `json:"entries/minute"`
}

func (co *CountOutput) logCount() error {
	now := time.Now()
	elapsedMinutes := math.Floor(now.Sub(co.start).Minutes())
	entriesPerMinute := float64(co.numEntries.Int64()) / math.Max(elapsedMinutes, 1)
	msg := &countObject{
		Entries:          co.numEntries,
		ElapsedMinutes:   elapsedMinutes,
		EntriesPerMinute: entriesPerMinute,
	}
	return co.encoder.Encode(msg)
}

func (co *CountOutput) determineOutput() error {
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
