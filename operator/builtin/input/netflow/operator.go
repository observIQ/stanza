package netflow

import (
	"context"
	"fmt"
	"net"
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

	NFLEnable bool   `json:"nfl_enable,omitempty" yaml:"nfl_enable,omitempty"`
	NFLAddr   string `json:"nfl_addr,omitempty"  yaml:"nfl_addr,omitempty"`
	NFLPort   uint   `json:"nfl_port,omitempty"  yaml:"nfl_port,omitempty"`
	NFLReuse  bool   `json:"nfl_reuse,omitempty"  yaml:"nfl_reuse,omitempty"`
}

// Build will build a netflow input operator.
func (c *NetflowInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.NFLEnable && c.NFLAddr == "" {
		return nil, fmt.Errorf("nfl_addr is a required value when nfl_enable is true")
	}

	if c.NFLAddr != "" {
		if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.NFLAddr, c.NFLPort)); err != nil {
			return nil, fmt.Errorf("failed to resolve nfl_addr: %s", err)
		}
	}

	netflowInput := &NetflowInput{
		InputOperator: inputOperator,
		nflEnable:     c.NFLEnable,
		nflAddr:       c.NFLAddr,
		nflPort:       c.NFLPort,
		nflReuse:      c.NFLReuse,
	}
	return []operator.Operator{netflowInput}, nil
}

// NetflowInput is an operator that receives netflow from network devices
type NetflowInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc

	// NetFlow v5
	nflEnable bool
	nflAddr   string
	nflPort   uint
	nflReuse  bool
}

// Start will start generating log entries.
func (n *NetflowInput) Start() error {
	startGoFlow(n, n.nflEnable, n.nflReuse, n.nflAddr, int(n.nflPort))

	return nil
}

// Stop will stop generating logs.
func (n *NetflowInput) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}
