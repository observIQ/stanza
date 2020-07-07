package parser

import (
	"context"

	"github.com/observiq/bplogagent/entry"
	"github.com/observiq/bplogagent/errors"
	"github.com/observiq/bplogagent/plugin"
	"github.com/observiq/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TimeParserConfig is the configuration of a time parser plugin.
type TimeParserConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	helper.TimeParser        `yaml:",omitempty,inline"`
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if err := c.TimeParser.Validate(context); err != nil {
		return nil, err
	}

	timeParser := &TimeParserPlugin{
		TransformerPlugin: transformerPlugin,
		TimeParser:        c.TimeParser,
	}

	return timeParser, nil
}

// TimeParserPlugin is a plugin that parses time from a field to an entry.
type TimeParserPlugin struct {
	helper.TransformerPlugin
	helper.TimeParser
}

// CanOutput will always return true for a parser plugin.
func (t *TimeParserPlugin) CanOutput() bool {
	return true
}

// Process will parse time from an entry.
func (t *TimeParserPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	if err := t.Parse(ctx, entry); err != nil {
		return errors.Wrap(err, "parse timestamp")
	}
	t.Write(ctx, entry)
	return nil
}
