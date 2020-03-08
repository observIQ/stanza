package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("rate_limit", &RateLimitConfig{})
}

type RateLimitConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
	Rate                      float64
	Interval                  float64
	Burst                     uint
}

func (c RateLimitConfig) Build(context pg.BuildContext) (pg.Plugin, error) {

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
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	plugin := &RateLimitPlugin{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,

		Interval: interval,
		Burst:    c.Burst,
	}

	return plugin, nil
}

type RateLimitPlugin struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	Interval time.Duration
	Burst    uint

	isReady chan struct{}
	cancel  context.CancelFunc
}

func (p *RateLimitPlugin) Input(entry *entry.Entry) error {
	<-p.isReady
	return p.Output(entry)
}

func (p *RateLimitPlugin) Start() error {
	p.isReady = make(chan struct{}, p.Burst)
	ticker := time.NewTicker(p.Interval)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	// Buffer the ticker ticks in isReady to allow bursts
	go func() {
		defer ticker.Stop()
		defer close(p.isReady)
		for {
			select {
			case <-ticker.C:
				p.isReady <- struct{}{}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (p *RateLimitPlugin) Stop() {
	p.cancel()
}
