package regex

import (
	"context"
	"sync"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/cache"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func newTestParser(t *testing.T, regex string) *RegexParser {
	cfg := NewRegexParserConfig("test")
	cfg.Regex = regex
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]
	return op.(*RegexParser)
}

func TestRegexParserBuildFailure(t *testing.T) {
	cfg := NewRegexParserConfig("test")
	cfg.OnError = "invalid_on_error"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestRegexParserStringFailure(t *testing.T) {
	parser := newTestParser(t, "^(?P<key>test)")
	_, err := parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "regex pattern does not match")
}

func TestRegexParserByteFailure(t *testing.T) {
	parser := newTestParser(t, "^(?P<key>test)")
	_, err := parser.parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "regex pattern does not match")
}

func TestRegexParserInvalidType(t *testing.T) {
	parser := newTestParser(t, "^(?P<key>test)")
	_, err := parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type '[]int' cannot be parsed as regex")
}

func TestParserRegex(t *testing.T) {
	cases := []struct {
		name         string
		configure    func(*RegexParserConfig)
		inputRecord  interface{}
		outputRecord interface{}
	}{
		{
			"RootString",
			func(p *RegexParserConfig) {
				p.Regex = "a=(?P<a>.*)"
			},
			"a=b",
			map[string]interface{}{
				"a": "b",
			},
		},
		{
			"RootBytes",
			func(p *RegexParserConfig) {
				p.Regex = "a=(?P<a>.*)"
			},
			[]byte("a=b"),
			map[string]interface{}{
				"a": "b",
			},
		},
		{
			"MemeoryCache",
			func(p *RegexParserConfig) {
				p.Regex = "a=(?P<a>.*)"
				p.CacheType = "memory"
			},
			"a=b",
			map[string]interface{}{
				"a": "b",
			},
		},
		{
			"K8sStringCache",
			func(p *RegexParserConfig) {
				p.Regex = `^(?P<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?P<namespace>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`
				p.CacheType = "memory"
			},
			[]byte("coredns-5644d7b6d9-mzngq_kube-system_coredns-901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6.log"),
			map[string]interface{}{
				"container_id":   "901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6",
				"container_name": "coredns",
				"namespace":      "kube-system",
				"pod_name":       "coredns-5644d7b6d9-mzngq",
			},
		},
		{
			"K8sBytesCache",
			func(p *RegexParserConfig) {
				p.Regex = `^(?P<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?P<namespace>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`
				p.CacheType = "memory"
			},
			"coredns-5644d7b6d9-mzngq_kube-system_coredns-901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6.log",
			map[string]interface{}{
				"container_id":   "901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6",
				"container_name": "coredns",
				"namespace":      "kube-system",
				"pod_name":       "coredns-5644d7b6d9-mzngq",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewRegexParserConfig("test")
			cfg.OutputIDs = []string{"fake"}
			tc.configure(cfg)

			ops, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			op := ops[0]

			fake := testutil.NewFakeOutput(t)
			op.SetOutputs([]operator.Operator{fake})

			// initial parse
			entry1 := entry.New()
			entry1.Record = tc.inputRecord
			err = op.Process(context.Background(), entry1)
			require.NoError(t, err)
			require.Equal(t, tc.outputRecord, entry1.Record)
			fake.ExpectRecord(t, tc.outputRecord)

			// parse identical input record a second time
			entry2 := entry.New()
			entry2.Record = tc.inputRecord
			err = op.Process(context.Background(), entry2)
			require.NoError(t, err)
			require.Equal(t, tc.outputRecord, entry2.Record, "expected cached entry to match input record")

			// compare both output records
			require.Equal(t, entry1.Record, entry2.Record, "expected cached entry to match initial entry")
		})
	}
}

func TestBuildParserRegex(t *testing.T) {
	newBasicRegexParser := func() *RegexParserConfig {
		cfg := NewRegexParserConfig("test")
		cfg.OutputIDs = []string{"test"}
		cfg.Regex = "(?P<all>.*)"
		return cfg
	}

	t.Run("BasicConfig", func(t *testing.T) {
		c := newBasicRegexParser()
		_, err := c.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
	})

	t.Run("MissingRegexField", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = ""
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
	})

	t.Run("InvalidRegexField", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = "())()"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
	})

	t.Run("NoNamedGroups", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = ".*"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no named capture groups")
	})

	t.Run("NoNamedGroups", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = "(.*)"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no named capture groups")
	})
}

const benchPattern = `^(?P<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?P<namespace>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`
const benchNumThreads = 10
const benchInput = "coredns-5644d7b6d9-mzngq_kube-system_coredns-901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6.log"

func BenchmarkParse(b *testing.B) {
	parser := newTestParser(&testing.T{}, benchPattern)
	parser.Cache = nil

	var n int
	var wg sync.WaitGroup

	for i := 0; i < benchNumThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n = 0; n < b.N; n++ {
				if _, err := parser.match(benchInput); err != nil {
					panic(err)
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkParseWithMemoryCache(b *testing.B) {
	parser := newTestParser(&testing.T{}, benchPattern)
	parser.Cache = cache.NewMemory(100)

	var n int
	var wg sync.WaitGroup

	for i := 0; i < benchNumThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n = 0; n < b.N; n++ {
				if _, err := parser.match(benchInput); err != nil {
					panic(err)
				}
			}
		}()
	}

	wg.Wait()
}
