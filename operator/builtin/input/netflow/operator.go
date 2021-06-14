package netflow

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/cloudflare/goflow/v3/utils"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	log "github.com/sirupsen/logrus"
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

	SFlowEnable bool   `json:"sflow_enable,omitempty" yaml:"sflow_enable,omitempty"`
	SFlowAddr   string `json:"sflow_addr,omitempty"   yaml:"sflow_addr,omitempty"`
	SFlowPort   uint   `json:"sflow_port,omitempty"   yaml:"sflow_port,omitempty"`
	SFlowReuse  bool   `json:"sflow_reuse,omitempty"  yaml:"sflow_reuse,omitempty"`

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

	if c.SFlowEnable && c.SFlowAddr == "" {
		return nil, fmt.Errorf("nfl_addr is a required value when nfl_enable is true")
	}

	if c.SFlowAddr != "" {
		if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.SFlowAddr, c.SFlowPort)); err != nil {
			return nil, err
		}
	}

	if c.NFLEnable && c.NFLAddr == "" {
		return nil, fmt.Errorf("nfl_addr is a required value when nfl_enable is true")
	}

	if c.NFLAddr != "" {
		if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.NFLAddr, c.NFLPort)); err != nil {
			return nil, err
	}

	netflowInput := &NetflowInput{
		InputOperator: inputOperator,
		sflowEnable:   c.SFlowEnable,
		sflowAddr:     c.SFlowAddr,
		sflowPort:     int(c.SFlowPort),
		sflowReuse:    c.SFlowReuse,
		nflEnable:     c.NFLEnable,
		nflAddr:       c.NFLAddr,
		nflPort:       int(c.NFLPort),
		nflReuse:      c.NFLReuse,
	}
	return []operator.Operator{netflowInput}, nil
}

// NetflowInput is an operator that receives netflow from network devices
type NetflowInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc

	// sFlow
	sflowEnable bool
	sflowAddr   string
	sflowPort   int
	sflowReuse  bool

	// NetFlow v5
	nflEnable bool
	nflAddr   string
	nflPort   int
	nflReuse  bool
}

// Start will start generating log entries.
func (n *NetflowInput) Start() error {
	/*startGoFlow(n, n.nflEnable, n.nflReuse, n.nflAddr, int(n.nflPort))
	return nil*/

	//runtime.GOMAXPROCS(runtime.NumCPU())

	//log.Info("Starting GoFlow")

	sSFlow := &utils.StateSFlow{
		Transport: n,
		Logger:    log.StandardLogger(), 
	}
	/*sNF := &utils.StateNetFlow{
		Transport: transport,
		Logger:    log.StandardLogger(), 
	}*/
	sNFL := &utils.StateNFLegacy{
		Transport: n,
		Logger:    log.StandardLogger(),
	}

	// TODO: This will expose prom metrics, do we want this?
	//go httpServer(sNF)

	//wg := &sync.WaitGroup{}
	if n.sflowEnable {
		n.Infof("starting sflow listener %s:%d", n.sflowAddr, n.sflowPort)
		err := sSFlow.FlowRoutine(*Workers, n.sflowAddr, n.sflowPort, n.sflowReuse)
		if err != nil {
			return err
		}
	}
	/*if *NFEnable {
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{
				"Type": "NetFlow"}).
				Infof("Listening on UDP %v:%v", *NFAddr, *NFPort)

			err := sNF.FlowRoutine(*Workers, *NFAddr, *NFPort, *NFReuse)
			if err != nil {
				log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
			}
			wg.Done()
		}()
	}*/
	if n.nflEnable {
		n.Infof("starting netflow v5 udp listener %s:%d", n.nflAddr, n.nflPort)
		err := sNFL.FlowRoutine(*Workers, n.nflAddr, n.nflPort, n.nflReuse)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop will stop generating logs.
func (n *NetflowInput) Stop() error {
	n.cancel()
	n.wg.Wait()
	return nil
}
