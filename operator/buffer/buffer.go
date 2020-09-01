package buffer

import (
	"context"
	"fmt"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
)

// Buffer is an entity that buffers log entries to an operator
type Buffer interface {
	Flush(context.Context) error
	Add(interface{}, int) error
	AddWait(context.Context, interface{}, int) error
	SetHandler(BundleHandler)
	Process(context.Context, *entry.Entry) error
}

// NewConfig creates a new buffer config
func NewConfig() Config {
	return Config{
		BufferType:           "memory",
		DelayThreshold:       operator.Duration{Duration: time.Second},
		BundleCountThreshold: 10_000,
		BundleByteThreshold:  4 * 1024 * 1024 * 1024,   // 4MB
		BundleByteLimit:      4 * 1024 * 1024 * 1024,   // 4MB
		BufferedByteLimit:    500 * 1024 * 1024 * 1024, // 500MB
		HandlerLimit:         16,
		Retry:                NewRetryConfig(),
	}
}

// Config is the configuration of a buffer
type Config struct {
	BufferType           string            `json:"type,omitempty"                   yaml:"type,omitempty"`
	DelayThreshold       operator.Duration `json:"delay_threshold,omitempty"        yaml:"delay_threshold,omitempty"`
	BundleCountThreshold int               `json:"bundle_count_threshold,omitempty" yaml:"buffer_count_threshold,omitempty"`
	BundleByteThreshold  int               `json:"bundle_byte_threshold,omitempty"  yaml:"bundle_byte_threshold,omitempty"`
	BundleByteLimit      int               `json:"bundle_byte_limit,omitempty"      yaml:"bundle_byte_limit,omitempty"`
	BufferedByteLimit    int               `json:"buffered_byte_limit,omitempty"    yaml:"buffered_byte_limit,omitempty"`
	HandlerLimit         int               `json:"handler_limit,omitempty"          yaml:"handler_limit,omitempty"`
	Retry                RetryConfig       `json:"retry,omitempty"                  yaml:"retry,omitempty"`
}

// Build will build a buffer from the supplied configuration
func (config *Config) Build() (Buffer, error) {
	switch config.BufferType {
	case "memory", "":
		return NewMemoryBuffer(config), nil
	default:
		return nil, errors.NewError(
			fmt.Sprintf("Invalid buffer type %s", config.BufferType),
			"The only supported buffer type is 'memory'",
		)
	}
}

// NewRetryConfig creates a new retry config
func NewRetryConfig() RetryConfig {
	return RetryConfig{
		InitialInterval:     operator.Duration{Duration: 500 * time.Millisecond},
		RandomizationFactor: 0.5,
		Multiplier:          1.5,
		MaxInterval:         operator.Duration{Duration: 15 * time.Minute},
	}
}

// RetryConfig is the configuration of an entity that will retry processing after an error
type RetryConfig struct {
	InitialInterval     operator.Duration `json:"initial_interval,omitempty"     yaml:"initial_interval,omitempty"`
	RandomizationFactor float64           `json:"randomization_factor,omitempty" yaml:"randomization_factor,omitempty"`
	Multiplier          float64           `json:"multiplier,omitempty"           yaml:"multiplier,omitempty"`
	MaxInterval         operator.Duration `json:"max_interval,omitempty"         yaml:"max_interval,omitempty"`
	MaxElapsedTime      operator.Duration `json:"max_elapsed_time,omitempty"     yaml:"max_elapsed_time,omitempty"`
}
