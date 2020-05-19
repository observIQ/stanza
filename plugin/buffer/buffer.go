package buffer

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

type Buffer interface {
	Flush(context.Context) error
	Add(interface{}, int) error
	AddWait(context.Context, interface{}, int) error
	SetHandler(func(context.Context, []*entry.Entry) error)
	Process(context.Context, *entry.Entry) error
}

type BufferConfig struct {
	BufferType           string          `json:"type,omitempty"                   yaml:"type,omitempty"`
	DelayThreshold       plugin.Duration `json:"delay_threshold,omitempty"        yaml:"delay_threshold,omitempty"`
	BundleCountThreshold int             `json:"bundle_count_threshold,omitempty" yaml:"buffer_count_threshold,omitempty"`
	BundleByteThreshold  int             `json:"bundle_byte_threshold,omitempty"  yaml:"bundle_byte_threshold,omitempty"`
	BundleByteLimit      int             `json:"bundle_byte_limit,omitempty"      yaml:"bundle_byte_limit,omitempty"`
	BufferedByteLimit    int             `json:"buffered_byte_limit,omitempty"    yaml:"buffered_byte_limit,omitempty"`
	HandlerLimit         int             `json:"handler_limit,omitempty"          yaml:"handler_limit,omitempty"`
	Retry                RetryConfig     `json:"retry,omitempty" yaml:"retry,omitempty"`
}

func (config *BufferConfig) Build() (Buffer, error) {
	config.setDefaults()

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

func (config *BufferConfig) setDefaults() {
	if config.BufferType == "" {
		config.BufferType = "memory"
	}

	if config.DelayThreshold.Raw() == time.Duration(0) {
		config.DelayThreshold = plugin.Duration{
			Duration: time.Second,
		}
	}

	if config.BundleCountThreshold == 0 {
		config.BundleCountThreshold = 10000
	}

	if config.BundleByteThreshold == 0 {
		config.BundleByteThreshold = 4 * 1024 * 1024 * 1024
	}

	if config.BundleByteLimit == 0 {
		config.BundleByteLimit = 4 * 1024 * 1024 * 1024
	}

	if config.BundleByteLimit == 0 {
		config.BundleByteLimit = 4 * 1024 * 1024 * 1024
	}

	if config.BufferedByteLimit == 0 {
		config.BufferedByteLimit = 500 * 1024 * 1024 * 1024 // 500MB
	}

	if config.HandlerLimit == 0 {
		config.HandlerLimit = 32
	}

	config.Retry.setDefaults()
}

type RetryConfig struct {
	InitialInterval     plugin.Duration
	RandomizationFactor float64
	Multiplier          float64
	MaxInterval         plugin.Duration
	MaxElapsedTime      plugin.Duration
}

func (config *RetryConfig) setDefaults() {
	if config.InitialInterval.Raw() == time.Duration(0) {
		config.InitialInterval = plugin.Duration{
			Duration: 500 * time.Millisecond,
		}
	}

	if config.RandomizationFactor == 0 {
		config.RandomizationFactor = 0.5
	}

	if config.Multiplier == 0 {
		config.Multiplier = 1.5
	}

	if config.MaxInterval.Raw() == time.Duration(0) {
		config.MaxInterval = plugin.Duration{
			Duration: 15 * time.Minute,
		}
	}
}
