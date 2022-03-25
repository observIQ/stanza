package otlp

import (
	"time"

	"github.com/observiq/stanza/operator/helper"
)

// GRPCClientSettings defines common settings for a gRPC client configuration.
type GRPCClientSettings struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
}

// RetrySettings defines retry settings for a gRPC client configuration.
type RetrySettings struct {
	// Enabled indicates whether to not retry sending batches in case of export failure.
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// Headers defines headers settings for a gRPC client configuration.
type Headers struct {
	Authorization string `json:"authorization" yaml:"authorization"`
}

// OtlpConfig is the configuration of an otlp output operation.
type OtlpConfig struct {
	helper.OutputConfig `yaml:",inline"`
	Endpoint            string `json:"endpoint" yaml:"endpoint"`
	Insecure            string `json:"insecure" yaml:"insecure"`
	Headers             `json:"headers" yaml:"headers"`
	RetrySettings       `json:"retry_on_failure" yaml:"retry_on_failure"`
	Timeout             time.Duration `json:"timeout" yaml:"timeout"`
}
