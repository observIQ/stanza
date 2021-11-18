package severity

import (
	"context"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
)

func init() {
	operator.Register("severity_parser", func() operator.Builder { return NewSeverityParserConfig("") })
}

// NewSeverityParserConfig creates a new severity parser config with default values
func NewSeverityParserConfig(operatorID string) *SeverityParserConfig {
	return &SeverityParserConfig{
		TransformerConfig:    helper.NewTransformerConfig(operatorID, "severity_parser"),
		SeverityParserConfig: helper.NewSeverityParserConfig(),
	}
}

// SeverityParserConfig is the configuration of a severity parser operator.
type SeverityParserConfig struct {
	helper.TransformerConfig    `yaml:",inline"`
	helper.SeverityParserConfig `yaml:",omitempty,inline"`
}

// Build will build a time parser operator.
func (c SeverityParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
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

	return []operator.Operator{severityOperator}, nil
}

// SeverityParserOperator is an operator that parses time from a field to an entry.
type SeverityParserOperator struct {
	helper.TransformerOperator
	helper.SeverityParser
}

// Process will parse time from an entry.
func (p *SeverityParserOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Parse)
}
