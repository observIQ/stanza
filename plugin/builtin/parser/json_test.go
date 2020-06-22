package parser

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/internal/testutil"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() (*JSONParser, *testutil.Plugin) {
	mock := testutil.Plugin{}
	logger, _ := zap.NewProduction()
	return &JSONParser{
		ParserPlugin: helper.ParserPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:      "test",
				PluginType:    "json_parser",
				SugaredLogger: logger.Sugar(),
			},
			Output:    &mock,
			ParseFrom: entry.NewField("testfield"),
			ParseTo:   entry.NewField("testparsed"),
		},
		json: jsoniter.ConfigFastest,
	}, &mock
}

func TestJSONImplementations(t *testing.T) {
	require.Implements(t, (*plugin.Plugin)(nil), new(JSONParser))
}

func TestJSONParser(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    map[string]interface{}
		expectedRecord map[string]interface{}
		errorExpected  bool
	}{
		{
			"simple",
			map[string]interface{}{
				"testfield": `{}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{},
			},
			false,
		},
		{
			"nested",
			map[string]interface{}{
				"testfield": `{"superkey":"superval"}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"superkey": "superval",
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.New()
			input.Record = tc.inputRecord

			output := entry.New()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeJSONPlugin()
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				require.Equal(t, tc.expectedRecord, e.Record)
			}).Return(nil)

			err := parser.Process(context.Background(), input)
			require.NoError(t, err)
		})
	}
}

func TestJSONParserWithEmbeddedTimeParser(t *testing.T) {

	testTime := time.Unix(1136214245, 0)

	cases := []struct {
		name           string
		inputRecord    map[string]interface{}
		expectedRecord map[string]interface{}
		errorExpected  bool
		preserve       bool
	}{
		{
			"simple",
			map[string]interface{}{
				"testfield": `{"timestamp":1136214245}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{},
			},
			false,
			false,
		},
		{
			"preserve",
			map[string]interface{}{
				"testfield": `{"timestamp":"1136214245"}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"timestamp": "1136214245",
				},
			},
			false,
			true,
		},
		{
			"nested",
			map[string]interface{}{
				"testfield": `{"superkey":"superval","timestamp":1136214245.123}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"superkey": "superval",
				},
			},
			false,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.New()
			input.Record = tc.inputRecord

			output := entry.New()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeJSONPlugin()
			parser.ParserPlugin.TimeParser = &helper.TimeParser{
				ParseFrom:  entry.NewField("testparsed", "timestamp"),
				LayoutType: "epoch",
				Layout:     "s",
				Preserve:   tc.preserve,
			}
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				require.Equal(t, tc.expectedRecord, e.Record)
				require.Equal(t, testTime, e.Timestamp)
			}).Return(nil)

			err := parser.Process(context.Background(), input)
			require.NoError(t, err)
		})
	}
}
