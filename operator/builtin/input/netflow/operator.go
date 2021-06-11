package netflow

import (
	"context"
	"sync"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

const operatorName = "netflow_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewNetflowInputConfig("") })
}

// NewNetflowInputConfig creates a new netflow input config with default values
func NewNetflowInputConfig(operatorID string) *NetflowInputConfig {
	return &NetflowInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
	}
}

// NetflowInputConfig is the configuration of a netflow input operator.
type NetflowInputConfig struct {
	helper.InputConfig `yaml:",inline"`
}

// Build will build a netflow input operator.
func (c *NetflowInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	netflowInput := &NetflowInput{
		InputOperator: inputOperator,
	}
	return []operator.Operator{netflowInput}, nil
}

// NetflowInput is an operator that receives netflow from network devices
type NetflowInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// Start will start generating log entries.
func (g *NetflowInput) Start() error {
	t := Transport{}
	startGoFlow(t)

	return nil
}

// Stop will stop generating logs.
func (g *NetflowInput) Stop() error {
	g.cancel()
	g.wg.Wait()
	return nil
}
