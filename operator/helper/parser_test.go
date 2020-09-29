package helper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestParserConfigMissingBase(t *testing.T) {
	config := ParserConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `type` field.")
}

func TestParserConfigInvalidTimeParser(t *testing.T) {
	cfg := NewParserConfig("test-id", "test-type")
	f := entry.NewRecordField("timestamp")
	cfg.TimeParser = &TimeParser{
		ParseFrom:  &f,
		Layout:     "",
		LayoutType: "strptime",
	}

	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required configuration parameter `layout`")
}

func TestParserConfigBuildValid(t *testing.T) {
	cfg := NewParserConfig("test-id", "test-type")
	f := entry.NewRecordField("timestamp")
	cfg.TimeParser = &TimeParser{
		ParseFrom:  &f,
		Layout:     "",
		LayoutType: "native",
	}
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}

func TestParserMissingField(t *testing.T) {
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: zaptest.NewLogger(t).Sugar(),
				},
			},
			OnError: DropOnError,
		},
		ParseFrom: entry.NewRecordField("test"),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Entry is missing the expected parse_from field.")
}

func TestParserInvalidParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
			},
			OnError: DropOnError,
		},
		ParseFrom: entry.NewRecordField(),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, fmt.Errorf("parse failure")
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse failure")
}

func TestParserInvalidTimeParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
			},
			OnError: DropOnError,
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
		TimeParser: &TimeParser{
			ParseFrom: func() *entry.Field {
				f := entry.NewRecordField("missing-key")
				return &f
			}(),
		},
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "time parser: log entry does not have the expected parse_from field")
}

func TestParserInvalidSeverityParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
			},
			OnError: DropOnError,
		},
		SeverityParser: &SeverityParser{
			ParseFrom: entry.NewRecordField("missing-key"),
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "severity parser: log entry does not have the expected parse_from field")
}

func TestParserInvalidTimeValidSeverityParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
			},
			OnError: DropOnError,
		},
		TimeParser: &TimeParser{
			ParseFrom: func() *entry.Field {
				f := entry.NewRecordField("missing-key")
				return &f
			}(),
		},
		SeverityParser: &SeverityParser{
			ParseFrom: entry.NewRecordField("severity"),
			Mapping: map[string]entry.Severity{
				"info": entry.Info,
			},
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := testEntry.Set(entry.NewRecordField("severity"), "info")
	require.NoError(t, err)

	err = parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "time parser: log entry does not have the expected parse_from field")

	// But, this should have been set anyways
	require.Equal(t, entry.Info, testEntry.Severity)
}

func TestParserValidTimeInvalidSeverityParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
			},
			OnError: DropOnError,
		},
		TimeParser: &TimeParser{
			ParseFrom: func() *entry.Field {
				f := entry.NewRecordField("timestamp")
				return &f
			}(),
			LayoutType: "gotime",
			Layout:     time.Kitchen,
		},
		SeverityParser: &SeverityParser{
			ParseFrom: entry.NewRecordField("missing-key"),
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := testEntry.Set(entry.NewRecordField("timestamp"), "12:34PM")
	require.NoError(t, err)

	err = parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "severity parser: log entry does not have the expected parse_from field")

	expected, _ := time.ParseInLocation(time.Kitchen, "12:34PM", time.Local)
	expected = setTimestampYear(expected)
	// But, this should have been set anyways
	require.Equal(t, expected, testEntry.Timestamp)
}

func TestParserOutput(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			OnError: DropOnError,
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
				OutputOperators: []operator.Operator{output},
			},
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.NoError(t, err)
	output.AssertCalled(t, "Process", mock.Anything, mock.Anything)
}

func TestParserWithPreserve(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			OnError: DropOnError,
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
				OutputOperators: []operator.Operator{output},
			},
		},
		ParseFrom: entry.NewRecordField("parse_from"),
		ParseTo:   entry.NewRecordField("parse_to"),
		Preserve:  true,
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := testEntry.Set(parser.ParseFrom, "test-value")
	require.NoError(t, err)

	err = parser.ProcessWith(ctx, testEntry, parse)
	require.NoError(t, err)

	actualValue, ok := testEntry.Get(parser.ParseFrom)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)

	actualValue, ok = testEntry.Get(parser.ParseTo)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)
}

func TestParserWithoutPreserve(t *testing.T) {
	output := &testutil.Operator{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserOperator{
		TransformerOperator: TransformerOperator{
			OnError: DropOnError,
			WriterOperator: WriterOperator{
				BasicOperator: BasicOperator{
					OperatorID:    "test-id",
					OperatorType:  "test-type",
					SugaredLogger: buildContext.Logger.SugaredLogger,
				},
				OutputOperators: []operator.Operator{output},
			},
		},
		ParseFrom: entry.NewRecordField("parse_from"),
		ParseTo:   entry.NewRecordField("parse_to"),
		Preserve:  false,
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := testEntry.Set(parser.ParseFrom, "test-value")
	require.NoError(t, err)
	err = parser.ProcessWith(ctx, testEntry, parse)
	require.NoError(t, err)

	_, ok := testEntry.Get(parser.ParseFrom)
	require.False(t, ok)

	actualValue, ok := testEntry.Get(parser.ParseTo)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)
}
