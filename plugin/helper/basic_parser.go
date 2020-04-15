package helper

import (
	"fmt"
	"regexp"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// BasicParserConfig provides the basic implementation of a parser config.
type BasicParserConfig struct {
	OutputID  string               `mapstructure:"output" yaml:"output"`
	ParseFrom *entry.FieldSelector `mapstructure:"parse_from" yaml:"parse_from"`
	ParseTo   *entry.FieldSelector `mapstructure:"parse_to" yaml:"parse_to"`
	OnError   string               `mapstructure:"on_error" yaml:"on_error"`
}

// Build will build a basic parser.
func (c BasicParserConfig) Build() (BasicParser, error) {
	if c.OutputID == "" {
		return BasicParser{}, fmt.Errorf("missing field 'output'")
	}

	if c.ParseFrom == nil {
		var fs entry.FieldSelector = entry.FieldSelector([]string{})
		c.ParseFrom = &fs
	}

	if c.ParseTo == nil {
		c.ParseTo = c.ParseFrom
	}

	if c.OnError == "" {
		c.OnError = "fail"
	}

	match, _ := regexp.Match(`^(fail|drop|ignore)$`, []byte(c.OnError))
	if !match {
		return BasicParser{}, fmt.Errorf("on_error must have a value of fail, drop, or ignore")
	}

	basicParser := BasicParser{
		OutputID:  c.OutputID,
		ParseFrom: *c.ParseFrom,
		ParseTo:   *c.ParseTo,
		OnError:   c.OnError,
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
		err := fmt.Errorf("parse_from field '%s' does not exist on the record", p.ParseFrom)
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
