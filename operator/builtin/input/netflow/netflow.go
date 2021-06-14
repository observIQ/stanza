package netflow

import "fmt"

// NetflowConfig is the configuration of a netflow input operator.
type NetflowConfig struct {
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
	Port    uint   `json:"port,omitempty"    yaml:"port,omitempty"`
	Workers uint   `json:"workers,omitempty" yaml:"workers,omitempty"`
}

func (c *NetflowConfig) Init() error {
	if c.Address == "" {
		c.Address = "0.0.0.0"
	}

	if c.Port == 0 {
		return fmt.Errorf("port is a required parameter")
	}

	if c.Workers == 0 {
		c.Workers = 1
	}

	return nil
}
