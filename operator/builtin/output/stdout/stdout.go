package output

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

// Stdout is a global handle to standard output
var Stdout io.Writer = os.Stdout

func init() {
	operator.Register("stdout", func() operator.Builder { return NewStdoutConfig("") })
}

func NewStdoutConfig(operatorID string) *StdoutConfig {
	return &StdoutConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "stdout"),
	}
}

// StdoutConfig is the configuration of the Stdout operator
type StdoutConfig struct {
	helper.OutputConfig `yaml:",inline"`
}

// Build will build a stdout operator.
func (c StdoutConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	return &StdoutOperator{
		OutputOperator: outputOperator,
		encoder:        json.NewEncoder(Stdout),
	}, nil
}

// StdoutOperator is an operator that logs entries using stdout.
type StdoutOperator struct {
	helper.OutputOperator
	encoder *json.Encoder
	mux     sync.Mutex
}

// Process will log entries received.
func (o *StdoutOperator) Process(ctx context.Context, entry *entry.Entry) error {
	o.mux.Lock()
	err := o.encoder.Encode(entry)
	if err != nil {
		o.mux.Unlock()
		o.Errorf("Failed to process entry: %s, $s", err, entry.Record)
		return err
	}
	o.mux.Unlock()
	return nil
}
