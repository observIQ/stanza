package parser

import (
	"context"

	"github.com/observiq/bplogagent/entry"
	"github.com/observiq/bplogagent/errors"
	"github.com/observiq/bplogagent/plugin"
	"github.com/observiq/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("severity_parser", &SeverityParserConfig{})
}

// SeverityParserConfig is the configuration of a severity parser plugin.
type SeverityParserConfig struct {
	helper.TransformerConfig    `yaml:",inline"`
	helper.SeverityParserConfig `yaml:",omitempty,inline"`
}

// Build will build a time parser plugin.
func (c SeverityParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	severityParser, err := c.SeverityParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	severityPlugin := &SeverityParserPlugin{
		TransformerPlugin: transformerPlugin,
		SeverityParser:    severityParser,
	}

	return severityPlugin, nil
}

// SeverityParserPlugin is a plugin that parses time from a field to an entry.
type SeverityParserPlugin struct {
	helper.TransformerPlugin
	helper.SeverityParser
}

// Process will parse time from an entry.
func (p *SeverityParserPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	if err := p.Parse(ctx, entry); err != nil {
		return errors.Wrap(err, "parse severity")
	}

	p.Write(ctx, entry)
	return nil
}
