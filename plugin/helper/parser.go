package helper

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
)

// ParserConfig provides the basic implementation of a parser config.
type ParserConfig struct {
	TransformerConfig `yaml:",inline"`

	ParseFrom            entry.Field           `json:"parse_from" yaml:"parse_from"`
	ParseTo              entry.Field           `json:"parse_to"   yaml:"parse_to"`
	Preserve             bool                  `json:"preserve"   yaml:"preserve"`
	TimeParser           *TimeParser           `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	SeverityParserConfig *SeverityParserConfig `json:"severity,omitempty" yaml:"severity,omitempty"`
}

// Build will build a parser plugin.
func (c ParserConfig) Build(context plugin.BuildContext) (ParserPlugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return ParserPlugin{}, err
	}

	if c.ParseFrom.FieldInterface == nil {
		c.ParseFrom.FieldInterface = entry.NewRecordField()
	}

	if c.ParseTo.FieldInterface == nil {
		c.ParseTo.FieldInterface = entry.NewRecordField()
	}

	if c.ParseFrom.String() == c.ParseTo.String() && c.Preserve {
		transformerPlugin.Warnw(
			"preserve is true, but parse_to is set to the same field as parse_from, "+
				"which will cause the original value to be overwritten",
			"plugin_id", c.ID(),
		)
	}

	parserPlugin := ParserPlugin{
		TransformerPlugin: transformerPlugin,
		ParseFrom:         c.ParseFrom,
		ParseTo:           c.ParseTo,
		Preserve:          c.Preserve,
	}

	if c.TimeParser != nil {
		if err := c.TimeParser.Validate(context); err != nil {
			return ParserPlugin{}, err
		}
		parserPlugin.TimeParser = c.TimeParser
	}

	if c.SeverityParserConfig != nil {
		severityParser, err := c.SeverityParserConfig.Build(context)
		if err != nil {
			return ParserPlugin{}, err
		}
		parserPlugin.SeverityParser = &severityParser
	}

	return parserPlugin, nil
}

// ParserPlugin provides a basic implementation of a parser plugin.
type ParserPlugin struct {
	TransformerPlugin
	ParseFrom      entry.Field
	ParseTo        entry.Field
	Preserve       bool
	TimeParser     *TimeParser
	SeverityParser *SeverityParser
}

// ProcessWith will process an entry with a parser function.
func (p *ParserPlugin) ProcessWith(ctx context.Context, entry *entry.Entry, parse ParseFunction) error {
	value, ok := entry.Get(p.ParseFrom)
	if !ok {
		err := errors.NewError(
			"Entry is missing the expected parse_from field.",
			"Ensure that all incoming entries contain the parse_from field.",
			"parse_from", p.ParseFrom.String(),
		)
		return p.HandleEntryError(ctx, entry, err)
	}

	newValue, err := parse(value)
	if err != nil {
		return p.HandleEntryError(ctx, entry, err)
	}

	if !p.Preserve {
		entry.Delete(p.ParseFrom)
	}

	entry.Set(p.ParseTo, newValue)

	var timeParseErr error
	if p.TimeParser != nil {
		timeParseErr = p.TimeParser.Parse(ctx, entry)
	}

	var severityParseErr error
	if p.SeverityParser != nil {
		severityParseErr = p.SeverityParser.Parse(ctx, entry)
	}

	// Handle time or severity parsing errors after attempting to parse both
	if timeParseErr != nil {
		return p.HandleEntryError(ctx, entry, errors.Wrap(timeParseErr, "time parser"))
	}
	if severityParseErr != nil {
		return p.HandleEntryError(ctx, entry, errors.Wrap(severityParseErr, "severity parser"))
	}

	p.Write(ctx, entry)
	return nil
}

// ParseFunction is function that parses a raw value.
type ParseFunction = func(interface{}) (interface{}, error)
