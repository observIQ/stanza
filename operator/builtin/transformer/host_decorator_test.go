package transformer

import (
	"context"
	"sync"
	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/stretchr/testify/require"
)

func TestHostDecorator(t *testing.T) {

	cases := []struct {
		name           string
		hd             *HostDecorator
		expectedLabels map[string]string
	}{
		{
			"Default",
			func() *HostDecorator {
				op, err := NewHostDecoratorConfig("").Build(testutil.NewBuildContext(t))
				require.NoError(t, err)
				hd := op.(*HostDecorator)
				hd.hostname = "test"
				return hd
			}(),
			map[string]string{
				"hostname": "test",
			},
		},
		{
			"NoHostname",
			func() *HostDecorator {
				cfg := NewHostDecoratorConfig("")
				cfg.IncludeHostname = false
				op, err := cfg.Build(testutil.NewBuildContext(t))
				require.NoError(t, err)
				hd := op.(*HostDecorator)
				hd.hostname = "test"
				return hd
			}(),
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := testutil.NewFakeOutput(t)
			tc.hd.OutputOperators = []operator.Operator{fake}
			e := entry.New()
			tc.hd.Process(context.Background(), e)
			select {
			case r := <-fake.Received:
				require.Equal(t, tc.expectedLabels, r.Labels)
			default:
				require.FailNow(t, "Expected entry")
			}
		})
	}
}

type hostDecoratorBenchmark struct {
	name   string
	cfgMod func(*HostDecoratorConfig)
}

func (g *hostDecoratorBenchmark) Run(b *testing.B) {
	cfg := NewHostDecoratorConfig(g.name)
	g.cfgMod(cfg)
	op, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)

	fake := testutil.NewFakeOutput(b)
	op.(*HostDecorator).OutputOperators = []operator.Operator{fake}

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

func BenchmarkGoogleCloudOutput(b *testing.B) {
	cases := []hostDecoratorBenchmark{
		{
			"Default",
			func(cfg *HostDecoratorConfig) {},
		},
		{
			"NoHostname",
			func(cfg *HostDecoratorConfig) {
				cfg.IncludeHostname = false
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, tc.Run)
	}
}
