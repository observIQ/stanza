package keyvalue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"go.uber.org/zap"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestParser(t *testing.T) *KVParser {
	config := NewKVParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	return op.(*KVParser)
}

func TestKVParserConfigBuild(t *testing.T) {
	config := NewKVParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	require.IsType(t, &KVParser{}, op)
}

func TestKVParserConfigBuildFailure(t *testing.T) {
	config := NewKVParserConfig("test")
	config.OnError = "invalid_on_error"
	_, err := config.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestBuild(t *testing.T) {
	basicConfig := func() *KVParserConfig {
		cfg := NewKVParserConfig("test_operator_id")
		return cfg
	}

	cases := []struct {
		name      string
		input     *KVParserConfig
		expectErr bool
	}{
		{
			"default",
			func() *KVParserConfig {
				cfg := basicConfig()
				return cfg
			}(),
			false,
		},
		{
			"delimiter",
			func() *KVParserConfig {
				cfg := basicConfig()
				cfg.Delimiter = "/"
				return cfg
			}(),
			false,
		},
		{
			"missing-delimiter",
			func() *KVParserConfig {
				cfg := basicConfig()
				cfg.Delimiter = ""
				return cfg
			}(),
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.input
			_, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestKVParserStringFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("expected '%s' to split by '%s' into two items, got", "invalid", parser.delimiter))
}

func TestKVParserByteFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("expected '%s' to split by '%s' into two items, got", "invalid", parser.delimiter))
}

func TestKVParserInvalidType(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type []int cannot be parsed as key value pairs")
}

func NewFakeKVOperator() (*KVParser, *testutil.Operator) {
	mock := testutil.Operator{}
	logger, _ := zap.NewProduction()
	return &KVParser{
		ParserOperator: helper.ParserOperator{
			TransformerOperator: helper.TransformerOperator{
				WriterOperator: helper.WriterOperator{
					BasicOperator: helper.BasicOperator{
						OperatorID:    "test",
						OperatorType:  "key_value_parser",
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

func TestKVImplementations(t *testing.T) {
	require.Implements(t, (*operator.Operator)(nil), new(KVParser))
}

func TestKVParser(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    map[string]interface{}
		expectedRecord map[string]interface{}
		delimiter      string
		errorExpected  bool
	}{
		{
			"simple",
			map[string]interface{}{
				"testfield": "name=stanza age=2",
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
				},
			},
			"=",
			false,
		},
		{
			"double-quotes-removed",
			map[string]interface{}{
				"testfield": "name=\"stanza\" age=2",
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
				},
			},
			"=",
			false,
		},
		{
			"double-quotes-spaces-removed",
			map[string]interface{}{
				"testfield": `name=" stanza " age=2`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
				},
			},
			"=",
			false,
		},
		{
			"leading-and-trailing-space",
			map[string]interface{}{
				"testfield": `" name "=" stanza " age=2`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
				},
			},
			"=",
			false,
		},
		{
			"bar-delimiter",
			map[string]interface{}{
				"testfield": `name|" stanza " age|2     key|value`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
					"key":  "value",
				},
			},
			"|",
			false,
		},
		{
			"double-delimiter",
			map[string]interface{}{
				"testfield": `name==" stanza " age==2     key==value`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"name": "stanza",
					"age":  "2",
					"key":  "value",
				},
			},
			"==",
			false,
		},
		{
			"bar-delimiter",
			map[string]interface{}{
				"testfield": `test/value a/b 2/text`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"test": "value",
					"a":    "b",
					"2":    "text",
				},
			},
			"/",
			false,
		},
		{
			"large",
			map[string]interface{}{
				"testfield": "name=stanza age=1 job=\"software engineering\" location=\"grand rapids michigan\" src=\"10.3.3.76\" dst=172.217.0.10 protocol=udp sport=57112 dport=443 translated_src_ip=96.63.176.3 translated_port=57112",
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"age":               "1",
					"dport":             "443",
					"dst":               "172.217.0.10",
					"job":               "software engineering",
					"location":          "grand rapids michigan",
					"name":              "stanza",
					"protocol":          "udp",
					"sport":             "57112",
					"src":               "10.3.3.76",
					"translated_port":   "57112",
					"translated_src_ip": "96.63.176.3",
				},
			},
			"=",
			false,
		},
		{
			"missing-delimiter",
			map[string]interface{}{
				"testfield": `test text`,
			},
			map[string]interface{}{},
			"/",
			true,
		},
		{
			"invalid-pair",
			map[string]interface{}{
				"testfield": `test=text=abc`,
			},
			map[string]interface{}{},
			"=",
			true,
		},
		{
			"empty-input",
			map[string]interface{}{
				"testfield": "",
			},
			map[string]interface{}{},
			"=",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.New()
			input.Record = tc.inputRecord

			output := entry.New()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeKVOperator()
			parser.delimiter = tc.delimiter
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				require.Equal(t, tc.expectedRecord, e.Record)
			}).Return(nil)

			err := parser.Process(context.Background(), input)
			if tc.errorExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestKVParserWithEmbeddedTimeParser(t *testing.T) {

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
				"testfield": "timestamp=1136214245",
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
				"testfield": "timestamp=1136214245",
			},
			map[string]interface{}{
				"testparsed":         map[string]interface{}{},
				"original_timestamp": "1136214245",
			},
			false,
			func() *entry.Field {
				f := entry.NewRecordField("original_timestamp")
				return &f
			}(),
		},
		{
			"preserve-multi-fields",
			map[string]interface{}{
				"testfield": "superkey=superval timestamp=1136214245",
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
			input.Record = tc.inputRecord

			output := entry.New()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeKVOperator()
			parser.delimiter = "="
			parseFrom := entry.NewRecordField("testparsed", "timestamp")
			parser.ParserOperator.TimeParser = &helper.TimeParser{
				ParseFrom:  &parseFrom,
				LayoutType: "epoch",
				Layout:     "s",
				PreserveTo: tc.preserveTo,
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

func TestSplitStringByWhitespace(t *testing.T) {
	cases := []struct {
		name   string
		intput string
		output []string
	}{
		{
			"simple",
			"k=v a=b x=\" y \" job=\"software engineering\"",
			[]string{
				"k=v",
				"a=b",
				"x=\" y \"",
				"job=\"software engineering\"",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, splitStringByWhitespace(tc.intput))
		})
	}
}

func BenchmarkParse(b *testing.B) {
	input := "name=stanza age=1 job=\"software engineering\" location=\"grand rapids michigan\" timestamp=1136214245 src=\"10.3.3.76\" dst=172.217.0.10 protocol=udp sport=57112 dport=443 translated_src_ip=96.63.176.3 translated_port=57112"

	kv := KVParser{
		delimiter: "=",
	}

	timeParseFrom := entry.NewRecordField("timestamp")
	kv.ParserOperator.TimeParser = &helper.TimeParser{
		ParseFrom:  &timeParseFrom,
		LayoutType: "epoch",
		Layout:     "s",
	}

	for n := 0; n < b.N; n++ {
		if _, err := kv.parse(input); err != nil {
			b.Fatal(err)
		}
	}
}
