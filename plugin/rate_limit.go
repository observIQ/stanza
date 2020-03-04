package plugin

import (
	"fmt"
	"sync"
	"time"
)

func init() {
	RegisterConfig("rate_limit", &RateLimitConfig{})
}

type RateLimitConfig struct {
	DefaultPluginConfig    `mapstructure:",squash"`
	DefaultOutputterConfig `mapstructure:",squash"`
	DefaultInputterConfig  `mapstructure:",squash"`
	Rate                   float64
	Interval               float64
	Burst                  uint64
}

func (c RateLimitConfig) Build(context BuildContext) (Plugin, error) {

	var interval time.Duration
	if c.Rate != 0 && c.Interval != 0 {
		return nil, fmt.Errorf("only one of 'rate' or 'interval' can be defined")
	} else if c.Rate < 0 || c.Interval < 0 {
		return nil, fmt.Errorf("rate and interval must be greater than zero")
	} else if c.Rate > 0 {
		interval = time.Second / time.Duration(c.Rate)
	}

	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to build default outputter: %s", err)
	}

	plugin := &RateLimitPlugin{
		DefaultPlugin:    defaultPlugin,
		DefaultInputter:  defaultInputter,
		DefaultOutputter: defaultOutputter,
		interval:         interval,
		burst:            c.Burst,
	}

	return plugin, nil
}

type RateLimitPlugin struct {
	DefaultPlugin
	DefaultOutputter
	DefaultInputter

	// Processed fields
	burst    uint64
	interval time.Duration
}

func (p *RateLimitPlugin) Start(wg *sync.WaitGroup) error {
	ticker := time.NewTicker(p.interval)

	go func() {
		defer wg.Done()
		defer ticker.Stop()

		isReady := make(chan struct{}, p.burst)
		exitTicker := make(chan struct{})
		defer close(exitTicker)

		// Buffer the ticker ticks to allow bursts
		go func() {
			for {
				select {
				case <-ticker.C:
					isReady <- struct{}{}
				case <-exitTicker:
					return
				}
			}
		}()

		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			<-isReady
			p.Output() <- entry
		}
	}()

	return nil
}
