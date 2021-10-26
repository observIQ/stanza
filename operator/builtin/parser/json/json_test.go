package json

import (
	"context"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestParser(t *testing.T) *JSONParser {
	config := NewJSONParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	return op.(*JSONParser)
}

func TestJSONParserConfigBuild(t *testing.T) {
	config := NewJSONParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	require.IsType(t, &JSONParser{}, op)
}

func TestJSONParserConfigBuildFailure(t *testing.T) {
	config := NewJSONParserConfig("test")
	config.OnError = "invalid_on_error"
	_, err := config.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestJSONParserStringFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error found in #1 byte")
}

func TestJSONParserByteFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "error found in #1 byte")
}

func TestJSONParserInvalidType(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type []int cannot be parsed as JSON")
}

func NewFakeJSONOperator() (*JSONParser, *testutil.Operator) {
	mock := testutil.Operator{}
	logger, _ := zap.NewProduction()
	return &JSONParser{
		ParserOperator: helper.ParserOperator{
			TransformerOperator: helper.TransformerOperator{
				WriterOperator: helper.WriterOperator{
					BasicOperator: helper.BasicOperator{
						OperatorID:    "test",
						OperatorType:  "json_parser",
						SugaredLogger: logger.Sugar(),
					},
					OutputOperators: []operator.Operator{&mock},
				},
			},
			ParseFrom: entry.NewBodyField("testfield"),
			ParseTo:   entry.NewBodyField("testparsed"),
		},
		json: jsoniter.ConfigFastest,
	}, &mock
}

func TestJSONImplementations(t *testing.T) {
	require.Implements(t, (*operator.Operator)(nil), new(JSONParser))
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
			input.Body = tc.inputRecord

			output := entry.New()
			output.Body = tc.expectedRecord

			parser, mockOutput := NewFakeJSONOperator()
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				require.Equal(t, tc.expectedRecord, e.Body)
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
		preserveTo     *entry.Field
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
			nil,
		},
		{
			"preserve",
			map[string]interface{}{
				"testfield": `{"timestamp":"1136214245"}`,
			},
			map[string]interface{}{
				"testparsed":         map[string]interface{}{},
				"original_timestamp": "1136214245",
			},
			false,
			func() *entry.Field {
				f := entry.NewBodyField("original_timestamp")
				return &f
			}(),
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
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.New()
			input.Body = tc.inputRecord

			output := entry.New()
			output.Body = tc.expectedRecord

			parser, mockOutput := NewFakeJSONOperator()
			parseFrom := entry.NewBodyField("testparsed", "timestamp")
			parser.ParserOperator.TimeParser = &helper.TimeParser{
				ParseFrom:  &parseFrom,
				LayoutType: "epoch",
				Layout:     "s",
				PreserveTo: tc.preserveTo,
			}
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				require.Equal(t, tc.expectedRecord, e.Body)
				require.Equal(t, testTime, e.Timestamp)
			}).Return(nil)

			err := parser.Process(context.Background(), input)
			require.NoError(t, err)
		})
	}
}
