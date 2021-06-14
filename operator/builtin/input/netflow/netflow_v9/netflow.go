package sflow

import (
	"context"
	"sync"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/netflow"
	"github.com/observiq/stanza/operator/helper"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

const operatorName = "netflow_v9_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewNetflowV9InputConfig("") })
}

// NewNetflowV9InputConfig creates a new netflow V9 input config with default values
func NewNetflowV9InputConfig(operatorID string) *NetflowV9nputConfig {
	return &NetflowV9nputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
	}
}

// NetflowV9nputConfig is the configuration of a netflow V9 input operator.
type NetflowV9nputConfig struct {
	helper.InputConfig    `yaml:",inline"`
	netflow.NetflowConfig `yaml:",inline"`
}

// Build will build a sflow input operator.
func (c *NetflowV9nputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if err := c.Init(); err != nil {
		return nil, err
	}

	netflowV9 := &NetflowV9Input{
		InputOperator: inputOperator,
		NetflowConfig: c.NetflowConfig,
	}
	return []operator.Operator{netflowV9}, nil
}

// NetflowV9Input is an operator that receives netflow V9 from network devices
type NetflowV9Input struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context

	netflow.NetflowConfig
}

// Start will start generating log entries.
func (n *NetflowV9Input) Start() error {
	_, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	flow := &utils.StateNetFlow{
		Transport: n,
		Logger:    log.StandardLogger(),
	}
	go func() {
		err := flow.FlowRoutine(int(n.Workers), n.Address, int(n.Port), n.Reuse)
		if err != nil {
			n.Errorf(err.Error())
		}
	}()

	return nil
}

// Stop will stop generating logs.
func (n *NetflowV9Input) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}

// Publish is required by GoFlows util.Transport interface
func (n NetflowV9Input) Publish(messages []*flowmessage.FlowMessage) {
	for _, msg := range messages {
		m, err := netflow.Parse(*msg)
		if err != nil {
			n.Errorf("Failed to parse sflow message", zap.Error(err))
		}

		entry, err := n.NewEntry(m)
		if err != nil {
			log.Error(err)
			continue
		}
		n.Write(n.ctx, entry)
	}
}
