package parser

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("severity_parser", func() plugin.Builder { return NewSeverityParserConfig("") })
}

func NewSeverityParserConfig(pluginID string) *SeverityParserConfig {
	return &SeverityParserConfig{
		TransformerConfig:    helper.NewTransformerConfig(pluginID, "severity_parser"),
		SeverityParserConfig: helper.NewSeverityParserConfig(),
	}
}

// SeverityParserConfig is the configuration of a severity parser plugin.
type SeverityParserConfig struct {
	helper.TransformerConfig    `yaml:",inline"`
	helper.SeverityParserConfig `yaml:",omitempty,inline"`
}

// Build will build a time parser plugin.
func (c SeverityParserConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	severityParser, err := c.SeverityParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	severityOperator := &SeverityParserOperator{
		TransformerOperator: transformerOperator,
		SeverityParser:      severityParser,
	}

	return severityOperator, nil
}

// SeverityParserOperator is a plugin that parses time from a field to an entry.
type SeverityParserOperator struct {
	helper.TransformerOperator
	helper.SeverityParser
}

// Process will parse time from an entry.
func (p *SeverityParserOperator) Process(ctx context.Context, entry *entry.Entry) error {
	if err := p.Parse(ctx, entry); err != nil {
		return errors.Wrap(err, "parse severity")
	}

	p.Write(ctx, entry)
	return nil
}
