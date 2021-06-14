package sflow

import (
	"testing"

	"github.com/observiq/stanza/operator/builtin/input/netflow"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name        string
		inputRecord SflowInputConfig
		expectErr   bool
	}{
		{
			"minimal",
			SflowInputConfig{
				NetflowConfig: netflow.NetflowConfig{
					Port: 2056,
				},
			},
			false,
		},
		{
			"missing-port",
			SflowInputConfig{
				NetflowConfig: netflow.NetflowConfig{
					Address: "0.0.0.0",
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewSflowInputConfig("test_id")

			if tc.inputRecord.Port > 0 {
				cfg.Port = tc.inputRecord.Port
			}

			_, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

}
