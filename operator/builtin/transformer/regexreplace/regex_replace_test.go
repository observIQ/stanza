package regex

import (
	"context"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
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

			entry := entry.New()
			entry.Record = tc.inputRecord
			err = op.Process(context.Background(), entry)
			require.NoError(t, err)

			fake.ExpectRecord(t, tc.outputRecord)
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
