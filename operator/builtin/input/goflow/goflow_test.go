package goflow

import (
	"testing"

	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name        string
		inputRecord GoflowInputConfig
		expectErr   bool
	}{
		{
			"minimal-netflow-v5",
			GoflowInputConfig{
				Mode: "netflow_v5",
				Port: 2056,
			},
			false,
		},
		{
			"minimal-netflow-v9",
			GoflowInputConfig{
				Mode: "netflow_v5",
				Port: 2056,
			},
			false,
		},
		{
			"minimal-netflow-sflow",
			GoflowInputConfig{
				Mode: "netflow_v5",
				Port: 2056,
			},
			false,
		},
		{
			"invalid mode",
			GoflowInputConfig{
				Mode: "netflow",
				Port: 2056,
			},
			true,
		},
		{
			"missing-port",
			GoflowInputConfig{
				Mode:    "sflow",
				Address: "0.0.0.0",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewGoflowInputConfig("test_id")
			cfg.Mode = tc.inputRecord.Mode
			cfg.Port = tc.inputRecord.Port

			_, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

}
