package netflowv5

import (
	"context"
	"sync"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/netflow"
	"github.com/observiq/stanza/operator/helper"
	log "github.com/sirupsen/logrus"
)

const operatorName = "netflow_v5_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewNetflowV5InputConfig("") })
}

// NewNetflowV5InputConfig creates a new netflow v5 input config with default values
func NewNetflowV5InputConfig(operatorID string) *NetflowV5InputConfig {
	return &NetflowV5InputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
	}
}

// NetflowV5InputConfig is the configuration of a netflow v5 input operator.
type NetflowV5InputConfig struct {
	helper.InputConfig    `yaml:",inline"`
	netflow.NetflowConfig `yaml:",inline"`
}

// Build will build a sflow input operator.
func (c *NetflowV5InputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if err := c.Init(); err != nil {
		return nil, err
	}

	netflowV5 := &NetflowV5Input{
		InputOperator: inputOperator,
		NetflowConfig: c.NetflowConfig,
	}
	return []operator.Operator{netflowV5}, nil
}

// NetflowV5Input is an operator that receives netflow v5 from network devices
type NetflowV5Input struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context

	netflow.NetflowConfig
}

// Start will start generating log entries.
func (n *NetflowV5Input) Start() error {
	_, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	flow := &utils.StateNFLegacy{
		Transport: n,
		Logger:    log.StandardLogger(),
	}
	go func() {
		err := flow.FlowRoutine(int(n.Workers), n.Address, int(n.Port), true)
		if err != nil {
			n.Errorf(err.Error())
		}
	}()

	return nil
}

// Stop will stop generating logs.
func (n *NetflowV5Input) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}

// Publish is required by GoFlows util.Transport interface
func (n NetflowV5Input) Publish(messages []*flowmessage.FlowMessage) {
	n.wg.Add(1)
	netflow.Publish(n.ctx, n.InputOperator, messages)
	n.wg.Done()
}
