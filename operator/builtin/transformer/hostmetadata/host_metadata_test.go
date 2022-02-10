package hostmetadata

import (
	"context"
	"sync"
	"testing"

	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/stretchr/testify/require"
)

type hostMetadataBenchmark struct {
	name   string
	cfgMod func(*HostMetadataConfig)
}

func (g *hostMetadataBenchmark) Run(b *testing.B) {
	cfg := NewHostMetadataConfig(g.name)
	g.cfgMod(cfg)
	ops, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)
	op := ops[0]

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
