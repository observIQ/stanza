package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("rate_filter", &RateFilterConfig{})
}

// RateFilterConfig is the configuration of a rate filter plugin.
type RateFilterConfig struct {
	base.FilterConfig `mapstructure:",squash" yaml:",inline"`
	Rate              float64
	Interval          float64
	Burst             uint
}

// Build will build a rate filter plugin.
func (c RateFilterConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	filterPlugin, err := c.FilterConfig.Build(context)
	if err != nil {
		return nil, err
	}

	var interval time.Duration
	if c.Rate != 0 && c.Interval != 0 {
		return nil, fmt.Errorf("only one of 'rate' or 'interval' can be defined")
	} else if c.Rate < 0 || c.Interval < 0 {
		return nil, fmt.Errorf("rate and interval must be greater than zero")
	} else if c.Rate > 0 {
		interval = time.Second / time.Duration(c.Rate)
	}

	rateLimitPlugin := &RateFilter{
		FilterPlugin: filterPlugin,
		Interval:     interval,
		Burst:        c.Burst,
	}

	return rateLimitPlugin, nil
}

// RateFilter is a plugin that limits the rate of log consumption between plugins.
type RateFilter struct {
	base.FilterPlugin
	Interval time.Duration
	Burst    uint

	isReady chan struct{}
	cancel  context.CancelFunc
}

// Consume will wait until a rate is met before sending an entry to the next plugin.
func (p *RateFilter) Consume(entry *entry.Entry) error {
	<-p.isReady
	return p.Output.Consume(entry)
}

// Start will start the rate limit plugin.
func (p *RateFilter) Start() error {
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

// Stop will stop the rate limit plugin.
func (p *RateFilter) Stop() error {
	p.cancel()
	return nil
}
