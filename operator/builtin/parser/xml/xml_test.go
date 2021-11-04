package xml

import (
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

func newTestParser(t *testing.T) *XMLParser {
	config := NewXMLParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	return op.(*XMLParser)
}

func TestXMLParserConfigBuild(t *testing.T) {
	config := NewXMLParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	require.IsType(t, &XMLParser{}, op)
}

func TestXMLParserConfigBuildFailure(t *testing.T) {
	config := NewXMLParserConfig("test")
	config.OnError = "invalid_on_error"
	_, err := config.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestXMLParserStringFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.Parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "CharData does not belong to any child")
}

func TestXMLParserStringFailure_v2(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.Parse(`<bookstore></bookstore>`)
	require.NoError(t, err)
	// require.Contains(t, err.Error(), "CharData does not belong to any child")
}

func TestXMLParserByteFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.Parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Value is not a string")
}

func TestXMLParserInvalidType(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.Parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Value is not a string")
}

func NewFakeXMLOperator() (*XMLParser, *testutil.Operator) {
	mock := testutil.Operator{}
	logger, _ := zap.NewProduction()
	return &XMLParser{
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
			ParseFrom: entry.NewRecordField("testfield"),
			ParseTo:   entry.NewRecordField("testparsed"),
		},
	}, &mock
}

func TestXMLImplementations(t *testing.T) {
	require.Implements(t, (*operator.Operator)(nil), new(XMLParser))
}

// func TestXMLParser(t *testing.T) {
// 	cases := []struct {
// 		name           string
// 		inputRecord    map[string]interface{}
// 		expectedRecord map[string]interface{}
// 		errorExpected  bool
// 	}{
// 		{
// 			"simple",
// 			map[string]interface{}{
// 				"testfield": `{}`,
// 			},
// 			map[string]interface{}{
// 				"testparsed": map[string]interface{}{},
// 			},
// 			false,
// 		},
// 		{
// 			"nested",
// 			map[string]interface{}{
// 				"testfield": `{"superkey":"superval"}`,
// 			},
// 			map[string]interface{}{
// 				"testparsed": map[string]interface{}{
// 					"superkey": "superval",
// 				},
// 			},
// 			false,
// 		},
// 	}

// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			input := entry.New()
// 			input.Record = tc.inputRecord

// 			output := entry.New()
// 			output.Record = tc.expectedRecord

// 			parser, mockOutput := NewFakeXMLOperator()
// 			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
// 				e := args[1].(*entry.Entry)
// 				require.Equal(t, tc.expectedRecord, e.Record)
// 			}).Return(nil)

// 			err := parser.Process(context.Background(), input)
// 			require.NoError(t, err)
// 		})
// 	}
// }

// func TestXMLParserWithEmbeddedTimeParser(t *testing.T) {

// 	testTime := time.Unix(1136214245, 0)

// 	cases := []struct {
// 		name           string
// 		inputRecord    map[string]interface{}
// 		expectedRecord map[string]interface{}
// 		errorExpected  bool
// 		preserveTo     *entry.Field
// 	}{
// 		{
// 			"simple",
// 			map[string]interface{}{
// 				"testfield": `{"timestamp":1136214245}`,
// 			},
// 			map[string]interface{}{
// 				"testparsed": map[string]interface{}{},
// 			},
// 			false,
// 			nil,
// 		},
// 		{
// 			"preserve",
// 			map[string]interface{}{
// 				"testfield": `{"timestamp":"1136214245"}`,
// 			},
// 			map[string]interface{}{
// 				"testparsed":         map[string]interface{}{},
// 				"original_timestamp": "1136214245",
// 			},
// 			false,
// 			func() *entry.Field {
// 				f := entry.NewRecordField("original_timestamp")
// 				return &f
// 			}(),
// 		},
// 		{
// 			"nested",
// 			map[string]interface{}{
// 				"testfield": `{"superkey":"superval","timestamp":1136214245.123}`,
// 			},
// 			map[string]interface{}{
// 				"testparsed": map[string]interface{}{
// 					"superkey": "superval",
// 				},
// 			},
// 			false,
// 			nil,
// 		},
// 	}

// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			input := entry.New()
// 			input.Record = tc.inputRecord

// 			output := entry.New()
// 			output.Record = tc.expectedRecord

// 			parser, mockOutput := NewFakeJSONOperator()
// 			parseFrom := entry.NewRecordField("testparsed", "timestamp")
// 			parser.ParserOperator.TimeParser = &helper.TimeParser{
// 				ParseFrom:  &parseFrom,
// 				LayoutType: "epoch",
// 				Layout:     "s",
// 				PreserveTo: tc.preserveTo,
// 			}
// 			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
// 				e := args[1].(*entry.Entry)
// 				require.Equal(t, tc.expectedRecord, e.Record)
// 				require.Equal(t, testTime, e.Timestamp)
// 			}).Return(nil)

// 			err := parser.Process(context.Background(), input)
// 			require.NoError(t, err)
// 		})
// 	}
// }
