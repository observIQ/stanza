package transformer

import (
	"context"
	"fmt"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("rate_limit", &RateLimitConfig{})
}

// RateLimitConfig is the configuration of a rate filter plugin.
type RateLimitConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Rate     float64         `json:"rate,omitempty"     yaml:"rate,omitempty"`
	Interval plugin.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
	Burst    uint            `json:"burst,omitempty"    yaml:"burst,omitempty"`
}

// Build will build a rate limit plugin.
func (c RateLimitConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	var interval time.Duration
	switch {
	case c.Rate != 0 && c.Interval.Raw() != 0:
		return nil, fmt.Errorf("only one of 'rate' or 'interval' can be defined")
	case c.Rate < 0 || c.Interval.Raw() < 0:
		return nil, fmt.Errorf("rate and interval must be greater than zero")
	case c.Rate > 0:
		interval = time.Second / time.Duration(c.Rate)
	default:
		interval = c.Interval.Raw()
	}

	rateLimitPlugin := &RateLimitPlugin{
		TransformerPlugin: transformerPlugin,
		interval:          interval,
		burst:             c.Burst,
	}

	return rateLimitPlugin, nil
}

// RateLimitPlugin is a plugin that limits the rate of log consumption between plugins.
type RateLimitPlugin struct {
	helper.TransformerPlugin

	interval time.Duration
	burst    uint
	isReady  chan struct{}
	cancel   context.CancelFunc
}

// Process will wait until a rate is met before sending an entry to the output.
func (p *RateLimitPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	<-p.isReady
	p.Write(ctx, entry)
	return nil
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
