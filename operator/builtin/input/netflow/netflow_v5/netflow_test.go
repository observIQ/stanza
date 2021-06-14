package netflowv5

import (
	"testing"

	"github.com/observiq/stanza/operator/builtin/input/netflow"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name        string
		inputRecord NetflowV5InputConfig
		expectErr   bool
	}{
		{
			"minimal",
			NetflowV5InputConfig{
				NetflowConfig: netflow.NetflowConfig{
					Port: 2056,
				},
			},
			false,
		},
		{
			"missing-port",
			NetflowV5InputConfig{
				NetflowConfig: netflow.NetflowConfig{
					Address: "0.0.0.0",
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewNetflowV5InputConfig("test_id")

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
