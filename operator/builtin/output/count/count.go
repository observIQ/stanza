package counter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

// CountOutputConfig is the configuration of a count output operator.
type CountOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	Path                string          `json:"path,omitempty" yaml:"path,omitempty"`
	Duration            helper.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
}

var defaultCounterDuration = helper.NewDuration(1 * time.Minute)

func init() {
	operator.Register("count_output", func() operator.Builder { return NewCounterOutputConfig("") })
}

// NewCounterOutputConfig creates the default config for the count_output operator.
func NewCounterOutputConfig(operatorID string) *CountOutputConfig {
	return &CountOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "count_output"),
		Duration:     defaultCounterDuration,
	}
}

// Build will build an instance of the count_output operator
func (c CountOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	counterOperator := &CountOutput{
		OutputOperator: outputOperator,
		ctx:            ctx,
		cancel:         cancel,
		interval:       c.Duration.Raw(),
		path:           c.Path,
		numEntries:     0,
		wg:             sync.WaitGroup{},
	}

	return []operator.Operator{counterOperator}, nil
}

// CountOutput is an output operator
type CountOutput struct {
	helper.OutputOperator
	ctx      context.Context
	interval time.Duration
	start    time.Time
	path     string
	file     *os.File
	encoder  *json.Encoder
	wg       sync.WaitGroup
	cancel   context.CancelFunc

	numEntries uint64
}

// Process increments the counter of the output operator
func (co *CountOutput) Process(_ context.Context, _ *entry.Entry) error {
	atomic.AddUint64(&co.numEntries, 1)
	return nil
}

// Start begins messaging count output to either stdout or a file
func (co *CountOutput) Start(_ operator.Persister) error {
	err := co.determineOutput()
	if err != nil {
		return err
	}

	co.start = time.Now()
	co.wg.Add(1)
	go co.startCounting()

	return nil
}

// Stop tells the CountOutput to stop gracefully
func (co *CountOutput) Stop() error {
	co.cancel()
	co.wg.Wait()
	if co.file != nil {
		return co.file.Close()
	}
	return nil
}

func (co *CountOutput) startCounting() {
	defer co.wg.Done()

	ticker := time.NewTicker(co.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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
	Entries          uint64  `json:"entries"`
	ElapsedMinutes   float64 `json:"elapsedMinutes"`
	EntriesPerMinute float64 `json:"entries/minute"`
	Timestamp        string  `json:"timestamp"`
}

func (co *CountOutput) logCount() error {
	now := time.Now()
	numEntries := atomic.LoadUint64(&co.numEntries)
	elapsedMinutes := now.Sub(co.start).Minutes()
	entriesPerMinute := float64(numEntries) / math.Max(elapsedMinutes, 1)
	msg := &countObject{
		Entries:          numEntries,
		ElapsedMinutes:   elapsedMinutes,
		EntriesPerMinute: entriesPerMinute,
		Timestamp:        now.Format(time.RFC3339),
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
	co.file = file
	co.encoder = json.NewEncoder(file)
	return nil
}
