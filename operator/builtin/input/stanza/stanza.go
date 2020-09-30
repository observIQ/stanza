package stanza

import (
	"context"
	"sync"

	"github.com/observiq/stanza/logger"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("stanza_input", func() operator.Builder { return NewInputConfig("") })
}

// NewInputConfig creates a new stanza input config with default values
func NewInputConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "stanza_input"),
		BufferSize:  100,
	}
}

// InputConfig is the configuration of a stanza input operator.
type InputConfig struct {
	helper.InputConfig `yaml:",inline"`
	BufferSize         int `json:"buffer_size" yaml:"buffer_size"`
}

// Build will build a stanza input operator.
func (c *InputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	receiver := make(logger.Receiver, c.BufferSize)
	context.Logger.AddReceiver(receiver)

	input := &Input{
		InputOperator: inputOperator,
		receiver:      receiver,
	}
	return input, nil
}

// Input is an operator that receives internal stanza logs.
type Input struct {
	helper.InputOperator

	receiver logger.Receiver
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

// Start will start reading incoming stanza logs.
func (i *Input) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel
	i.startReading(ctx)
	return nil
}

// Stop will stop reading logs.
func (i *Input) Stop() error {
	i.cancel()
	i.wg.Wait()
	return nil
}

// startReading will start reading stanza logs from the receiver.
func (i *Input) startReading(ctx context.Context) {
	i.wg.Add(1)
	go func() {
		defer i.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-i.receiver:
				i.Write(ctx, &e)
			}
		}
	}()
}
