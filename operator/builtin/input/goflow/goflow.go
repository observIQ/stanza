package goflow

import (
	"context"
	"fmt"
	"sync"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	log "github.com/sirupsen/logrus"
)

const operatorName = "goflow_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewGoflowInputConfig("") })
}

// NewGoflowInputConfig creates a new goflow input config with default values
func NewGoflowInputConfig(operatorID string) *GoflowInputConfig {
	return &GoflowInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
	}
}

// GoflowInputConfig is the configuration of a goflow input operator.
type GoflowInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Mode    string `json:"mode,omitempty"    yaml:"mode,omitempty"`
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
	Port    uint   `json:"port,omitempty"    yaml:"port,omitempty"`
	Workers uint   `json:"workers,omitempty" yaml:"workers,omitempty"`
}

// Build will build a goflow input operator.
func (c *GoflowInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	switch c.Mode {
	case "sflow", "netflow_v5", "netflow_v9":
		break
	default:
		return nil, fmt.Errorf("%s is not a supported Goflow mode", c.Mode)
	}

	if c.Address == "" {
		c.Address = "0.0.0.0"
	}

	if c.Port == 0 {
		return nil, fmt.Errorf("port is a required parameter")
	}

	if c.Workers == 0 {
		c.Workers = 1
	}

	goflowInput := &GoflowInput{
		InputOperator: inputOperator,
		mode:          c.Mode,
		address:       c.Address,
		port:          int(c.Port),
		workers:       int(c.Workers),
	}
	return []operator.Operator{goflowInput}, nil
}

// GoflowInput is an operator that receives network traffic information
// from network devices.
type GoflowInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context

	mode    string
	address string
	port    int
	workers int
}

// Start will start generating log entries.
func (n *GoflowInput) Start() error {
	_, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	reuse := true

	switch n.mode {
	case "sflow":
		flow := &utils.StateSFlow{
			Transport: n,
			Logger:    log.StandardLogger(),
		}
		go func() {
			err := flow.FlowRoutine(n.workers, n.address, n.port, reuse)
			if err != nil {
				n.Errorf(err.Error())
			}
		}()

	case "netflow_v5":
		flow := &utils.StateNFLegacy{
			Transport: n,
			Logger:    log.StandardLogger(),
		}
		go func() {
			err := flow.FlowRoutine(n.workers, n.address, n.port, reuse)
			if err != nil {
				n.Errorf(err.Error())
			}
		}()

	case "netflow_v9":
		flow := &utils.StateNetFlow{
			Transport: n,
			Logger:    log.StandardLogger(),
		}
		go func() {
			err := flow.FlowRoutine(n.workers, n.address, n.port, reuse)
			if err != nil {
				n.Errorf(err.Error())
			}
		}()

	default:
		return fmt.Errorf("%s is not a supported Goflow mode", n.mode)
	}

	n.Infof("Started Goflow on %s:%d in %s mode", n.address, n.port, n.mode)
	return nil
}

// Stop will stop generating logs.
func (n *GoflowInput) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}

// Publish wraps WriteGoFlowMessage and is required by GoFlows util.Transport interface
func (n GoflowInput) Publish(messages []*flowmessage.FlowMessage) {
	n.WriteGoFlowMessage(n.ctx, messages)
}
