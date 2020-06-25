package parser

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
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

	return p.Output.Process(ctx, entry)
}
