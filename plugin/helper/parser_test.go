package helper

import (
	"context"
	"fmt"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParserConfigMissingBase(t *testing.T) {
	config := ParserConfig{}
	context := testutil.NewBuildContext(t)
	_, err := config.Build(context)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Plugin config is missing the `id` field.")
}

func TestParserConfigInvalidTimeParser(t *testing.T) {
	config := ParserConfig{
		TransformerConfig: TransformerConfig{
			BasicConfig: BasicConfig{
				PluginID:   "test-id",
				PluginType: "test-type",
			},
			OutputID: "test-output",
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
			OutputID: "test-output",
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
		ParseFrom: entry.NewField("test"),
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
		TimeParser: &TimeParser{
			ParseFrom: entry.NewField("missing-key"),
		},
	}
	parse := func(i interface{}) (interface{}, error) {
		return i, nil
	}
	ctx := context.Background()
	testEntry := entry.New()
	err := parser.ProcessWith(ctx, testEntry, parse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "log entry does not have the expected parse_from field")
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
			Output:  output,
		},
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
			Output:  output,
		},
		ParseFrom: entry.NewField("parse_from"),
		ParseTo:   entry.NewField("parse_to"),
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
			Output:  output,
		},
		ParseFrom: entry.NewField("parse_from"),
		ParseTo:   entry.NewField("parse_to"),
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

	actualValue, ok := testEntry.Get(parser.ParseFrom)
	require.False(t, ok)

	actualValue, ok = testEntry.Get(parser.ParseTo)
	require.True(t, ok)
	require.Equal(t, "test-value", actualValue)
}
