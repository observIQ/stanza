package transformer

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/stretchr/testify/require"
)

func testHostname() (string, error) {
	return "test", nil
}

func TestHostMetadata(t *testing.T) {
	cases := []struct {
		name           string
		modifyConfig   func(*HostMetadataConfig)
		expectedLabels map[string]string
		fakeHostname   func() (string, error)
	}{
		{
			"Default",
			func(cfg *HostMetadataConfig) {
				cfg.IncludeIP = false
			},
			map[string]string{
				"hostname": "test",
			},
			testHostname,
		},
		{
			"NoHostname",
			func(cfg *HostMetadataConfig) {
				cfg.IncludeHostname = false
				cfg.IncludeIP = false
			},
			nil,
			testHostname,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hostname = tc.fakeHostname
			defer func() { hostname = os.Hostname }()

			cfg := NewHostMetadataConfig("test_id")
			cfg.OutputIDs = []string{"fake"}
			tc.modifyConfig(cfg)

			op, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)

			fake := testutil.NewFakeOutput(t)
			err = op.SetOutputs([]operator.Operator{fake})
			require.NoError(t, err)

			e := entry.New()
			op.Process(context.Background(), e)
			select {
			case r := <-fake.Received:
				require.Equal(t, tc.expectedLabels, r.Labels)
			default:
				require.FailNow(t, "Expected entry")
			}
		})
	}
}

type hostMetadataBenchmark struct {
	name   string
	cfgMod func(*HostMetadataConfig)
}

func (g *hostMetadataBenchmark) Run(b *testing.B) {
	cfg := NewHostMetadataConfig(g.name)
	g.cfgMod(cfg)
	op, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)

	fake := testutil.NewFakeOutput(b)
	op.(*HostMetadata).OutputOperators = []operator.Operator{fake}

	b.ResetTimer()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			e := entry.New()
			op.Process(context.Background(), e)
		}
		err = op.Stop()
		require.NoError(b, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			<-fake.Received
		}
	}()

	wg.Wait()
}

func BenchmarkHostMetadata(b *testing.B) {
	cases := []hostMetadataBenchmark{
		{
			"Default",
			func(cfg *HostMetadataConfig) {},
		},
		{
			"NoHostname",
			func(cfg *HostMetadataConfig) {
				cfg.IncludeHostname = false
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, tc.Run)
	}
}
