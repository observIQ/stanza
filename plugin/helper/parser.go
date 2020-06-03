package helper

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// ParserConfig provides the basic implementation of a parser config.
type ParserConfig struct {
	BasicConfig `yaml:",inline"`

	OutputID  string      `json:"output"     yaml:"output"`
	ParseFrom entry.Field `json:"parse_from" yaml:"parse_from"`
	ParseTo   entry.Field `json:"parse_to"   yaml:"parse_to"`
	Preserve  bool        `json:"preserve"   yaml:"preserve"`
	OnError   string      `json:"on_error"   yaml:"on_error"`
}

// ID will return the plugin id.
func (c ParserConfig) ID() string {
	return c.PluginID
}

// Type will return the plugin type.
func (c ParserConfig) Type() string {
	return c.PluginType
}

// Build will build a parser plugin.
func (c ParserConfig) Build(context plugin.BuildContext) (ParserPlugin, error) {
	basicPlugin, err := c.BasicConfig.Build(context)
	if err != nil {
		return ParserPlugin{}, err
	}

	if c.OutputID == "" {
		return ParserPlugin{}, errors.NewError(
			"Plugin config is missing the `output` field.",
			"Ensure that a valid `output` field exists on the plugin config.",
		)
	}

	if c.OnError == "" {
		c.OnError = "ignore"
	}

	switch c.OnError {
	case "fail", "drop", "ignore":
	default:
		return ParserPlugin{}, errors.NewError(
			"Plugin config has an invalid `on_error` field.",
			"Ensure that the `on_error` field is set to fail, drop, or ignore.",
			"on_error", c.OnError,
		)
	}

	parserPlugin := ParserPlugin{
		BasicPlugin: basicPlugin,
		OutputID:    c.OutputID,
		ParseFrom:   c.ParseFrom,
		ParseTo:     c.ParseTo,
		Preserve:    c.Preserve,
		OnError:     c.OnError,
	}

	return parserPlugin, nil
}

// SetNamespace will namespace the id and output of the plugin config.
func (c *ParserConfig) SetNamespace(namespace string, exclusions ...string) {
	if CanNamespace(c.PluginID, exclusions) {
		c.PluginID = AddNamespace(c.PluginID, namespace)
	}

	if CanNamespace(c.OutputID, exclusions) {
		c.OutputID = AddNamespace(c.OutputID, namespace)
	}
}

// ParserPlugin provides a basic implementation of a parser plugin.
type ParserPlugin struct {
	BasicPlugin
	OutputID  string
	ParseFrom entry.Field
	ParseTo   entry.Field
	Preserve  bool
	OnError   string
	Output    plugin.Plugin
}

// CanProcess will always return true for a parser plugin.
func (p *ParserPlugin) CanProcess() bool {
	return true
}

// CanOutput will always return true for a parser plugin.
func (p *ParserPlugin) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (p *ParserPlugin) Outputs() []plugin.Plugin {
	return []plugin.Plugin{p.Output}
}

// SetOutputs will set the output plugin.
func (p *ParserPlugin) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, p.OutputID)
	if err != nil {
		return err
	}

	p.Output = output
	return nil
}

// ProcessWith will process an entry with a parser function and forward the results to the output plugin.
func (p *ParserPlugin) ProcessWith(ctx context.Context, entry *entry.Entry, parseFunc ParseFunction) error {
	value, ok := entry.Get(p.ParseFrom)
	if !ok {
		err := errors.NewError(
			"Log entry does not have the expected parse_from field.",
			"Ensure that all entries forwarded to this parser contain the parse_from field.",
			"parse_from", p.ParseFrom.String(),
		)
		return p.HandleParserError(ctx, entry, err)
	}

	newValue, err := parseFunc(value)
	if err != nil {
		return p.HandleParserError(ctx, entry, err)
	}

	if !p.Preserve {
		entry.Delete(p.ParseFrom)
	}

	entry.Set(p.ParseTo, newValue)
	return p.Output.Process(ctx, entry)
}

// HandleParserError will handle an error based on the `OnError` property
func (p *ParserPlugin) HandleParserError(ctx context.Context, entry *entry.Entry, err error) error {
	p.Warnw("Failed to parse entry", zap.Any("error", err), "entry", entry)

	if p.OnError == "fail" {
		return err
	}

	if p.OnError == "drop" {
		return nil
	}

	return p.Output.Process(ctx, entry)
}

// ParseFunction is function that parses a raw value.
type ParseFunction = func(interface{}) (interface{}, error)
