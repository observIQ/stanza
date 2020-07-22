package transformer

import (
	"context"
	"fmt"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("rate_limit", func() operator.Builder { return NewRateLimitConfig("") })
}

func NewRateLimitConfig(operatorID string) *RateLimitConfig {
	return &RateLimitConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "rate_limit"),
	}
}

// RateLimitConfig is the configuration of a rate filter operator.
type RateLimitConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Rate     float64           `json:"rate,omitempty"     yaml:"rate,omitempty"`
	Interval operator.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
	Burst    uint              `json:"burst,omitempty"    yaml:"burst,omitempty"`
}

// Build will build a rate limit operator.
func (c RateLimitConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
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

	rateLimitOperator := &RateLimitOperator{
		TransformerOperator: transformerOperator,
		interval:            interval,
		burst:               c.Burst,
	}

	return rateLimitOperator, nil
}

// RateLimitOperator is an operator that limits the rate of log consumption between operators.
type RateLimitOperator struct {
	helper.TransformerOperator

	interval time.Duration
	burst    uint
	isReady  chan struct{}
	cancel   context.CancelFunc
}

// Process will wait until a rate is met before sending an entry to the output.
func (p *RateLimitOperator) Process(ctx context.Context, entry *entry.Entry) error {
	<-p.isReady
	p.Write(ctx, entry)
	return nil
}

// Start will start the rate limit operator.
func (p *RateLimitOperator) Start() error {
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

// Stop will stop the rate limit operator.
func (p *RateLimitOperator) Stop() error {
	p.cancel()
	return nil
}
