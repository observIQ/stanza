package regex

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func NewTestConfig(t *testing.T, regex string) (*operator.Config, error) {
	json := `{
		"type": "regex_parser",
		"id": "test_id",
		"regex": "%s",
		"output": "test_output"
	}`
	json = fmt.Sprintf(json, regex)
	config := &operator.Config{}
	err := config.UnmarshalJSON([]byte(json))
	return config, err
}

func NewTestParser(t *testing.T, regex string) (*RegexParser, error) {
	config, err := NewTestConfig(t, regex)
	if err != nil {
		return nil, err
	}

	ctx := testutil.NewBuildContext(t)
	op, err := config.Build(ctx)
	if err != nil {
		return nil, err
	}

	parser, ok := op.(*RegexParser)
	if !ok {
		return nil, fmt.Errorf("operator is not a regex parser")
	}

	return parser, nil
}

func TestRegexParserBuildFailure(t *testing.T) {
	config, err := NewTestConfig(t, "^(?P<key>test)")
	require.NoError(t, err)

	parserConfig, ok := config.Builder.(*RegexParserConfig)
	require.True(t, ok)

	parserConfig.OnError = "invalid_on_error"
	ctx := testutil.NewBuildContext(t)
	_, err = config.Build(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestRegexParserStringFailure(t *testing.T) {
	parser, err := NewTestParser(t, "^(?P<key>test)")
	require.NoError(t, err)

	_, err = parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "regex pattern does not match")
}

func TestRegexParserByteFailure(t *testing.T) {
	parser, err := NewTestParser(t, "^(?P<key>test)")
	require.NoError(t, err)

	_, err = parser.parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "regex pattern does not match")
}

func TestRegexParserInvalidType(t *testing.T) {
	parser, err := NewTestParser(t, "^(?P<key>test)")
	require.NoError(t, err)

	_, err = parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type '[]int' cannot be parsed as regex")
}

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
