package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("rate_limit", &RateLimitConfig{})
}

// RateLimitConfig is the configuration of a rate filter plugin.
type RateLimitConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	Rate     float64         `mapstructure:"rate"     json:"rate,omitempty"     yaml:"rate,omitempty"`
	Interval plugin.Duration `mapstructure:"interval" json:"interval,omitempty" yaml:"interval,omitempty"`
	Burst    uint            `mapstructure:"burst"    json:"burst,omitempty"    yaml:"burst,omitempty"`
}

// Build will build a rate limit plugin.
func (c RateLimitConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	var interval time.Duration
	if c.Rate != 0 && c.Interval.Raw() != 0 {
		return nil, fmt.Errorf("only one of 'rate' or 'interval' can be defined")
	} else if c.Rate < 0 || c.Interval.Raw() < 0 {
		return nil, fmt.Errorf("rate and interval must be greater than zero")
	} else if c.Rate > 0 {
		interval = time.Second / time.Duration(c.Rate)
	} else {
		interval = c.Interval.Raw()
	}

	rateLimitPlugin := &RateLimitPlugin{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,
		interval:         interval,
		burst:            c.Burst,
	}

	return rateLimitPlugin, nil
}

// RateLimitPlugin is a plugin that limits the rate of log consumption between plugins.
type RateLimitPlugin struct {
	helper.BasicPlugin
	helper.BasicTransformer

	interval time.Duration
	burst    uint
	isReady  chan struct{}
	cancel   context.CancelFunc
}

// Process will wait until a rate is met before sending an entry to the output.
func (p *RateLimitPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	<-p.isReady
	return p.Output.Process(ctx, entry)
}

// Start will start the rate limit plugin.
func (p *RateLimitPlugin) Start() error {
	p.isReady = make(chan struct{}, p.burst)
	ticker := time.NewTicker(p.interval)

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

// Stop will stop the rate limit plugin.
func (p *RateLimitPlugin) Stop() error {
	p.cancel()
	return nil
}
