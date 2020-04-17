package helper

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// BasicParserConfig provides the basic implementation of a parser config.
type BasicParserConfig struct {
	OutputID  string               `mapstructure:"output" yaml:"output"`
	ParseFrom *entry.FieldSelector `mapstructure:"parse_from" yaml:"parse_from"`
	ParseTo   *entry.FieldSelector `mapstructure:"parse_to" yaml:"parse_to"`
	OnError   string               `mapstructure:"on_error" yaml:"on_error"`
}

// Build will build a basic parser.
func (c BasicParserConfig) Build(logger *zap.SugaredLogger) (BasicParser, error) {
	if c.OutputID == "" {
		return BasicParser{}, errors.NewError(
			"Plugin config is missing the `output` field.",
			"Ensure that a valid `output` field exists on the plugin config.",
		)
	}

	if c.ParseFrom == nil {
		var fs entry.FieldSelector = entry.FieldSelector([]string{})
		c.ParseFrom = &fs
	}

	if c.ParseTo == nil {
		c.ParseTo = c.ParseFrom
	}

	if c.OnError == "" {
		c.OnError = "ignore"
	}

	switch c.OnError {
	case "fail", "drop", "ignore":
	default:
		return BasicParser{}, errors.NewError(
			"Plugin config has an invalid `on_error` field.",
			"Ensure that the `on_error` field is set to fail, drop, or ignore.",
			"on_error", c.OnError,
		)
	}

	basicParser := BasicParser{
		OutputID:      c.OutputID,
		ParseFrom:     *c.ParseFrom,
		ParseTo:       *c.ParseTo,
		OnError:       c.OnError,
		SugaredLogger: logger,
	}

	return basicParser, nil
}

// BasicParser provides a basic implementation of a parser plugin.
type BasicParser struct {
	OutputID  string
	ParseFrom entry.FieldSelector
	ParseTo   entry.FieldSelector
	OnError   string
	Output    plugin.Plugin
	*zap.SugaredLogger
}

// CanProcess will always return true for a parser plugin.
func (p *BasicParser) CanProcess() bool {
	return true
}

// CanOutput will always return true for a parser plugin.
func (p *BasicParser) CanOutput() bool {
	return true
}

// Outputs will return an array containing the output plugin.
func (p *BasicParser) Outputs() []plugin.Plugin {
	return []plugin.Plugin{p.Output}
}

// SetOutputs will set the output plugin.
func (p *BasicParser) SetOutputs(plugins []plugin.Plugin) error {
	output, err := FindOutput(plugins, p.OutputID)
	if err != nil {
		return err
	}

	p.Output = output
	return nil
}

// ProcessWith will process an entry with a parser function and forward the results to the output plugin.
func (p *BasicParser) ProcessWith(entry *entry.Entry, parseFunc ParseFunction) error {
	value, ok := entry.Get(p.ParseFrom)
	if !ok {
		err := errors.NewError(
			"Log entry does not have the expected parse_from field.",
			"Ensure that all entries forwarded to this parser contain the parse_from field.",
		)
		return p.HandleParserError(entry, err)
	}

	newValue, err := parseFunc(value)
	if err != nil {
		return p.HandleParserError(entry, err)
	}

	entry.Set(p.ParseTo, newValue)
	return p.Output.Process(entry)
}

// HandleParserError will handle an error based on the `OnError` property
func (p *BasicParser) HandleParserError(entry *entry.Entry, err error) error {
	p.Errorw("Failed to parse entry", zap.Any("error", err))

	if p.OnError == "fail" {
		return err
	}

	if p.OnError == "drop" {
		return nil
	}

	return p.Output.Process(entry)
}

// ParseFunction is function that parses a raw value.
type ParseFunction = func(interface{}) (interface{}, error)
