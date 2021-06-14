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

const operatorName = "sflow_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewSflowInputConfig("") })
}

// NewSflowInputConfig creates a new sflow input config with default values
func NewSflowInputConfig(operatorID string) *SflowInputConfig {
	return &SflowInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
	}
}

// SflowInputConfig is the configuration of a sflow input operator.
type SflowInputConfig struct {
	helper.InputConfig    `yaml:",inline"`
	netflow.NetflowConfig `yaml:",inline"`
}

// Build will build a sflow input operator.
func (c *SflowInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if err := c.Init(); err != nil {
		return nil, err
	}

	sflowInput := &SflowInput{
		InputOperator: inputOperator,
		NetflowConfig: c.NetflowConfig,
	}
	return []operator.Operator{sflowInput}, nil
}

// SflowInput is an operator that receives sflow from network devices
type SflowInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context

	netflow.NetflowConfig
}

// Start will start generating log entries.
func (n *SflowInput) Start() error {
	_, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	flow := &utils.StateSFlow{
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
func (n *SflowInput) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}

// Publish is required by GoFlows util.Transport interface
func (n SflowInput) Publish(messages []*flowmessage.FlowMessage) {
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
