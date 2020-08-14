package helper

import (
	"fmt"
	"net"
	"os"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
)

// NewHostLabelerConfig returns a HostLabelerConfig with default values
func NewHostLabelerConfig() HostLabelerConfig {
	return HostLabelerConfig{
		IncludeHostname: true,
		IncludeIP:       true,
		getHostname:     getHostname,
		getIP:           getIP,
	}
}

// HostLabelerConfig is the configuration of a host labeler
type HostLabelerConfig struct {
	IncludeHostname bool `json:"include_hostname,omitempty"     yaml:"include_hostname,omitempty"`
	IncludeIP       bool `json:"include_ip,omitempty"     yaml:"include_ip,omitempty"`
	getHostname     func() (string, error)
	getIP           func() (string, error)
}

// Build will build a host labeler from the supplied configuration
func (c HostLabelerConfig) Build() (HostLabeler, error) {
	labeler := HostLabeler{
		includeHostname: c.IncludeHostname,
		includeIP:       c.IncludeIP,
	}

	if c.getHostname == nil {
		return labeler, fmt.Errorf("getHostname func is not set")
	}

	if c.getIP == nil {
		return labeler, fmt.Errorf("getIP func is not set")
	}

	if c.IncludeHostname {
		hostname, err := c.getHostname()
		if err != nil {
			return labeler, errors.Wrap(err, "get hostname")
		}
		labeler.hostname = hostname
	}

	if c.IncludeIP {
		ip, err := c.getIP()
		if err != nil {
			return labeler, errors.Wrap(err, "get ip address")
		}
		labeler.ip = ip
	}

	return labeler, nil
}

// getHostname will return the hostname of the current host
func getHostname() (string, error) {
	return os.Hostname()
}

// getIP will return the IP address of the current host
func getIP() (string, error) {
	var ip string

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "list interfaces")
	}

	for _, i := range interfaces {
		// Skip loopback interfaces
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip down interfaces
		if i.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := i.Addrs()
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

// HostLabeler is a helper that adds host related metadata to an entry's labels
type HostLabeler struct {
	hostname        string
	ip              string
	includeHostname bool
	includeIP       bool
}

// Label will label an entry with host related metadata
func (h *HostLabeler) Label(entry *entry.Entry) {
	if h.includeHostname {
		entry.AddLabel("hostname", h.hostname)
	}

	if h.includeIP {
		entry.AddLabel("ip", h.ip)
	}
}
