package transformer

import (
	"context"
	"net"
	"os"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("host_metadata", func() operator.Builder { return NewHostMetadataConfig("") })
}

// Variables that are overridable for testing
var hostname = os.Hostname

// NewHostMetadataConfig returns a HostMetadataConfig with default values
func NewHostMetadataConfig(operatorID string) *HostMetadataConfig {
	return &HostMetadataConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "host_decorator"),
		IncludeHostname:   true,
		IncludeIP:         true,
	}
}

//
type HostMetadataConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	IncludeHostname          bool `json:"include_hostname,omitempty"     yaml:"include_hostname,omitempty"`
	IncludeIP                bool `json:"include_ip,omitempty"     yaml:"include_ip,omitempty"`
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
		includeIP:           c.IncludeIP,
	}

	if c.IncludeHostname {
		op.hostname, err = hostname()
		if err != nil {
			return nil, errors.Wrap(err, "get hostname")
		}
	}

	if c.IncludeIP {
		ip, err := getIP()
		if err != nil {
			return nil, errors.Wrap(err, "get ip address")
		}
		op.ip = ip
	}

	return op, nil
}

func getIP() (string, error) {
	var ip string

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "list interfaces")
	}

	for _, iface := range ifaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		if len(addrs) > 0 {
			ip = addrs[0].String()
		}
	}

	if len(ip) == 0 {
		return "", errors.NewError(
			"failed to find ip address",
			"check that a non-loopback interface with an assigned IP address exists and is running",
		)
	}

	return ip, nil
}

// HostMetadata is an operator that can add host metadata to incoming entries
type HostMetadata struct {
	helper.TransformerOperator

	hostname        string
	ip              string
	includeHostname bool
	includeIP       bool
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

	if h.includeIP {
		entry.AddLabel("ip", h.ip)
	}

	return entry, nil
}
