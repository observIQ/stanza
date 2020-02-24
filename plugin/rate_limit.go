package plugin

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

func init() {
	RegisterConfig("rate_limit", &RateLimitConfig{})
}

type RateLimitConfig struct {
	DefaultProcessorConfig `mapstructure:",squash"`
	Rate                   float64
	Interval               float64
	Burst                  uint64
}

func (c RateLimitConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {

	var interval time.Duration
	if c.Rate != 0 && c.Interval != 0 {
		return nil, fmt.Errorf("only one of 'rate' or 'interval' can be defined")
	} else if c.Rate < 0 || c.Interval < 0 {
		return nil, fmt.Errorf("rate and interval must be greater than zero")
	} else if c.Rate > 0 {
		interval = time.Second / time.Duration(c.Rate)
	}

	plugin := &RateLimitPlugin{
		DefaultProcessor: c.DefaultProcessorConfig.Build(),
		config:           c,
		SugaredLogger:    logger.With("plugin_type", "json", "plugin_id", c.DefaultProcessorConfig.ID()),
		interval:         interval,
	}

	return plugin, nil
}

type RateLimitPlugin struct {
	DefaultProcessor
	config RateLimitConfig
	*zap.SugaredLogger

	// Processed fields
	interval time.Duration
}

func (p *RateLimitPlugin) Start(wg *sync.WaitGroup) error {
	ticker := time.NewTicker(p.interval)

	go func() {
		defer wg.Done()
		defer ticker.Stop()

		isReady := make(chan struct{}, p.config.Burst)
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
