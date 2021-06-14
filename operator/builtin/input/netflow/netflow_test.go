package netflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	cases := []struct {
		name        string
		input       NetflowConfig
		expectError bool
	}{
		{
			"minimal",
			NetflowConfig{
				Port: 2056,
			},
			false,
		},
		{
			"all",
			NetflowConfig{
				Address: "10.1.1.1",
				Port:    2056,
				Reuse:   false,
				Workers: 10,
			},
			false,
		},
		{
			"missing-port",
			NetflowConfig{
				Address: "10.1.1.1",
				Reuse:   false,
				Workers: 10,
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectError {
				require.Error(t, tc.input.Init())
			} else {
				require.NoError(t, tc.input.Init())
			}
		})
	}
}
