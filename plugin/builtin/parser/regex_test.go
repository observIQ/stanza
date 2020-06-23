package parser

import (
	"context"
	"regexp"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newFakeRegexParser() (*RegexParser, *testutil.Plugin) {
	mockPlugin := testutil.Plugin{}
	return &RegexParser{
		ParserPlugin: helper.ParserPlugin{
			TransformerPlugin: helper.TransformerPlugin{
				BasicPlugin: helper.BasicPlugin{
					PluginID:   "regex_parser",
					PluginType: "regex_parser",
				},
				OutputID: "mock_output",
				Output:   &mockPlugin,
			},
		},
	}, &mockPlugin
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
	newBasicRegexParser := func() RegexParserConfig {
		return RegexParserConfig{
			ParserConfig: helper.ParserConfig{
				TransformerConfig: helper.TransformerConfig{
					BasicConfig: helper.BasicConfig{
						PluginID:   "test",
						PluginType: "test",
					},
					OutputID: "test",
				},
			},
			Regex: ".*",
		}
	}

	t.Run("BasicConfig", func(t *testing.T) {
		c := newBasicRegexParser()
		buildContext := plugin.BuildContext{
			Logger: zaptest.NewLogger(t).Sugar(),
		}
		_, err := c.Build(buildContext)
		require.NoError(t, err)
	})

	t.Run("MissingRegexField", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = ""
		buildContext := plugin.BuildContext{
			Logger: zaptest.NewLogger(t).Sugar(),
		}
		_, err := c.Build(buildContext)
		require.Error(t, err)
	})

	t.Run("InvalidRegexField", func(t *testing.T) {
		c := newBasicRegexParser()
		c.Regex = "())()"
		buildContext := plugin.BuildContext{
			Logger: zaptest.NewLogger(t).Sugar(),
		}
		_, err := c.Build(buildContext)
		require.Error(t, err)
	})
}
