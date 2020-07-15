package helper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParserConfigMissingBase(t *testing.T) {
	config := ParserConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required `id` field.")
}

func TestParserConfigInvalidTimeParser(t *testing.T) {
	config := ParserConfig{
		TransformerConfig: TransformerConfig{
			BasicConfig: BasicConfig{
				PluginID:   "test-id",
				PluginType: "test-type",
			},
			WriterConfig: WriterConfig{
				OutputIDs: []string{"test-output"},
			},
		},
		TimeParser: &TimeParser{
			Layout:     "",
			LayoutType: "strptime",
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required configuration parameter `layout`")
}

func TestParserConfigBuildValid(t *testing.T) {
	config := ParserConfig{
		TransformerConfig: TransformerConfig{
			BasicConfig: BasicConfig{
				PluginID:   "test-id",
				PluginType: "test-type",
			},
			WriterConfig: WriterConfig{
				OutputIDs: []string{"test-output"},
			},
		},
		TimeParser: &TimeParser{
			Layout:     "",
			LayoutType: "native",
		},
	}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.NoError(t, err)
}

func TestParserMissingField(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
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
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
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
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
		},
		ParseFrom: entry.NewRecordField(),
		ParseTo:   entry.NewRecordField(),
		TimeParser: &TimeParser{
			ParseFrom: entry.NewRecordField("missing-key"),
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
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
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
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
		},
		TimeParser: &TimeParser{
			ParseFrom: entry.NewRecordField("missing-key"),
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
	testEntry.Set(entry.NewRecordField("severity"), "info")

	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "time parser: log entry does not have the expected parse_from field")

	// But, this should have been set anyways
	require.Equal(t, entry.Info, testEntry.Severity)
}

func TestParserValidTimeInvalidSeverityParse(t *testing.T) {
	buildContext := testutil.NewBuildContext(t)
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
		},
		TimeParser: &TimeParser{
			ParseFrom:  entry.NewRecordField("timestamp"),
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
	testEntry.Set(entry.NewRecordField("timestamp"), "12:34PM")

	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "severity parser: log entry does not have the expected parse_from field")

	expected, _ := time.ParseInLocation(time.Kitchen, "12:34PM", time.Local)
	expected = setTimestampYear(expected)
	// But, this should have been set anyways
	require.Equal(t, expected, testEntry.Timestamp)
}

func TestParserOutput(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
			WriterPlugin: WriterPlugin{
				OutputPlugins: []plugin.Plugin{output},
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
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
			WriterPlugin: WriterPlugin{
				OutputPlugins: []plugin.Plugin{output},
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
	testEntry.Set(parser.ParseFrom, "test-value")
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.NoError(t, err)

	actualValue, ok := testEntry.Get(parser.ParseFrom)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)

	actualValue, ok = testEntry.Get(parser.ParseTo)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)
}

func TestParserWithoutPreserve(t *testing.T) {
	output := &testutil.Plugin{}
	output.On("ID").Return("test-output")
	output.On("Process", mock.Anything, mock.Anything).Return(nil)
	buildContext := testutil.NewBuildContext(t)
	parser := ParserPlugin{
		TransformerPlugin: TransformerPlugin{
			BasicPlugin: BasicPlugin{
				PluginID:      "test-id",
				PluginType:    "test-type",
				SugaredLogger: buildContext.Logger,
			},
			OnError: DropOnError,
			WriterPlugin: WriterPlugin{
				OutputPlugins: []plugin.Plugin{output},
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
	testEntry.Set(parser.ParseFrom, "test-value")
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.NoError(t, err)

	_, ok := testEntry.Get(parser.ParseFrom)
	require.False(t, ok)

	actualValue, ok := testEntry.Get(parser.ParseTo)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)
}
