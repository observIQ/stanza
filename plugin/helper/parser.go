package helper

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// ParserConfig provides the basic implementation of a parser config.
type ParserConfig struct {
	TransformerConfig `yaml:",inline"`

	ParseFrom  entry.Field `json:"parse_from" yaml:"parse_from"`
	ParseTo    entry.Field `json:"parse_to"   yaml:"parse_to"`
	Preserve   bool        `json:"preserve"   yaml:"preserve"`
	TimeParser *TimeParser `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
}

// Build will build a parser plugin.
func (c ParserConfig) Build(context plugin.BuildContext) (ParserPlugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return ParserPlugin{}, err
	}

	if c.TimeParser != nil {
		if err := c.TimeParser.Validate(context); err != nil {
			return ParserPlugin{}, err
		}
	}

	parserPlugin := ParserPlugin{
		TransformerPlugin: transformerPlugin,
		ParseFrom:         c.ParseFrom,
		ParseTo:           c.ParseTo,
		Preserve:          c.Preserve,
		TimeParser:        c.TimeParser,
	}

	return parserPlugin, nil
}

// ParserPlugin provides a basic implementation of a parser plugin.
type ParserPlugin struct {
	TransformerPlugin
	ParseFrom  entry.Field
	ParseTo    entry.Field
	Preserve   bool
	TimeParser *TimeParser
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

	if p.TimeParser != nil {
		if err := p.TimeParser.Parse(ctx, entry); err != nil {
			return p.HandleEntryError(ctx, entry, err)
		}
	}

	p.Write(ctx, entry)
	return nil
}

// ParseFunction is function that parses a raw value.
type ParseFunction = func(interface{}) (interface{}, error)
