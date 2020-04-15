package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() (*JSONParser, *testutil.Plugin) {
	mock := testutil.Plugin{}
	logger, _ := zap.NewProduction()
	return &JSONParser{
		BasicPlugin: helper.BasicPlugin{
			PluginID:      "test",
			PluginType:    "json_parser",
			SugaredLogger: logger.Sugar(),
		},
		BasicParser: helper.BasicParser{
			Output:    &mock,
			ParseFrom: entry.FieldSelector([]string{"testfield"}),
			ParseTo:   entry.FieldSelector([]string{"testparsed"}),
		},
		json: jsoniter.ConfigFastest,
	}, &mock
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(JSONParser))
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
				"testfield":  `{}`,
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
				"testfield": `{"superkey":"superval"}`,
				"testparsed": map[string]interface{}{
					"superkey": "superval",
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.NewEntry()
			input.Record = tc.inputRecord

			output := entry.NewEntry()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeJSONPlugin()
			mockOutput.On("Process", mock.Anything).Run(func(args mock.Arguments) {
				e := args[0].(*entry.Entry)
				if !assert.Equal(t, tc.expectedRecord, e.Record) {
					t.FailNow()
				}
			}).Return(nil)

			err := parser.Process(input)
			if !assert.NoError(t, err) {
				return
			}
		})
	}
}
