package transformer

import (
	"context"
	"os"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("host_metadata", func() operator.Builder { return NewHostMetadataConfig("") })
}

// NewHostMetadataConfig returns a HostMetadataConfig with default values
func NewHostMetadataConfig(operatorID string) *HostMetadataConfig {
	return &HostMetadataConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "host_decorator"),
		IncludeHostname:   true,
	}
}

//
type HostMetadataConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	IncludeHostname          bool `json:"include_hostname,omitempty"     yaml:"include_hostname,omitempty"`
}

// Build will build an operator from the supplied configuration
func (c HostMetadataConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build transformer")
	}

	op := &HostMetadata{
		TransformerOperator: transformerOperator,
		includeHostname:     c.IncludeHostname,
	}

	if c.IncludeHostname {
		op.hostname, err = os.Hostname()
		if err != nil {
			return nil, errors.Wrap(err, "get hostname")
		}
	}

	return op, nil
}

// HostMetadata is an operator that can add host metadata to incoming entries
type HostMetadata struct {
	helper.TransformerOperator

	hostname        string
	includeHostname bool
}

// Process will process an incoming entry using the metadata transform.
func (h *HostMetadata) Process(ctx context.Context, entry *entry.Entry) error {
	return h.ProcessWith(ctx, entry, h.Transform)
}

// Transform will transform an entry, adding the configured host metadata.
func (h *HostMetadata) Transform(entry *entry.Entry) (*entry.Entry, error) {
	if h.includeHostname {
		entry.AddLabel("hostname", h.hostname)
	}

	return entry, nil
}
