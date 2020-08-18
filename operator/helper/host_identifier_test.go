package helper

import (
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/stretchr/testify/require"
)

func MockHostIdentifierConfig(includeIP, includeHostname bool, ip, hostname string) HostIdentifierConfig {
	return HostIdentifierConfig{
		IncludeIP:       includeIP,
		IncludeHostname: includeHostname,
		getIP:           func() (string, error) { return ip, nil },
		getHostname:     func() (string, error) { return hostname, nil },
	}
}

func TestHostLabeler(t *testing.T) {
	cases := []struct {
		name             string
		config           HostIdentifierConfig
		expectedResource map[string]string
	}{
		{
			"HostnameAndIP",
			MockHostIdentifierConfig(true, true, "ip", "hostname"),
			map[string]string{
				"hostname": "hostname",
				"ip":       "ip",
			},
		},
		{
			"HostnameNoIP",
			MockHostIdentifierConfig(false, true, "ip", "hostname"),
			map[string]string{
				"hostname": "hostname",
			},
		},
		{
			"IPNoHostname",
			MockHostIdentifierConfig(true, false, "ip", "hostname"),
			map[string]string{
				"ip": "ip",
			},
		},
		{
			"NoHostnameNoIP",
			MockHostIdentifierConfig(false, false, "", "test"),
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			identifier, err := tc.config.Build()
			require.NoError(t, err)

			e := entry.New()
			identifier.Identify(e)
			require.Equal(t, tc.expectedResource, e.Resource)
		})
	}
}
