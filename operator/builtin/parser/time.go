package parser

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("time_parser", func() operator.Builder { return NewTimeParserConfig("") })
}

func NewTimeParserConfig(operatorID string) *TimeParserConfig {
	return &TimeParserConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "time_parser"),
		TimeParser:        helper.NewTimeParser(),
	}
}

// TimeParserConfig is the configuration of a time parser operator.
type TimeParserConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	helper.TimeParser        `yaml:",omitempty,inline"`
}

// Build will build a time parser operator.
func (c TimeParserConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if err := c.TimeParser.Validate(context); err != nil {
		return nil, err
	}

	timeParser := &TimeParserOperator{
		TransformerOperator: transformerOperator,
		TimeParser:          c.TimeParser,
	}

	return timeParser, nil
}

// TimeParserOperator is an operator that parses time from a field to an entry.
type TimeParserOperator struct {
	helper.TransformerOperator
	helper.TimeParser
}

// CanOutput will always return true for a parser operator.
func (t *TimeParserOperator) CanOutput() bool {
	return true
}

// Process will parse time from an entry.
func (t *TimeParserOperator) Process(ctx context.Context, entry *entry.Entry) error {
	if err := t.Parse(ctx, entry); err != nil {
		return errors.Wrap(err, "parse timestamp")
	}
	t.Write(ctx, entry)
	return nil
}
