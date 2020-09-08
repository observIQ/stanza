package regex

import (
	"context"
	"regexp"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newFakeRegexParser() (*RegexParser, *testutil.Operator) {
	mockOperator := testutil.Operator{}
	return &RegexParser{
		ParserOperator: helper.ParserOperator{
			TransformerOperator: helper.TransformerOperator{
				WriterOperator: helper.WriterOperator{
					BasicOperator: helper.BasicOperator{
						OperatorID:   "regex_parser",
						OperatorType: "regex_parser",
					},
					OutputIDs:       []string{"mock_output"},
					OutputOperators: []operator.Operator{&mockOperator},
				},
			},
			ParseFrom: entry.NewRecordField(),
			ParseTo:   entry.NewRecordField(),
		},
	}, &mockOperator
}

func TestParserRegex(t *testing.T) {
	cases := []struct {
		name         string
		configure    func(*RegexParser)
		inputRecord  interface{}
		outputRecord interface{}
	}{
		{
			"RootString",
			func(p *RegexParser) {
				p.regexp = regexp.MustCompile("a=(?P<a>.*)")
			},
			"a=b",
			map[string]interface{}{
				"a": "b",
			},
		},
		{
			"RootBytes",
			func(p *RegexParser) {
				p.regexp = regexp.MustCompile("a=(?P<a>.*)")
			},
			[]byte("a=b"),
			map[string]interface{}{
				"a": "b",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser, mockOutput := newFakeRegexParser()
			tc.configure(parser)

			var parsedRecord interface{}
			mockOutput.On("Process", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				parsedRecord = args.Get(1).(*entry.Entry).Record
			})

			entry := entry.Entry{
				Record: tc.inputRecord,
			}
			err := parser.Process(context.Background(), &entry)
			require.NoError(t, err)

			require.Equal(t, tc.outputRecord, parsedRecord)

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
