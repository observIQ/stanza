package builtin

import (
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TODO make this fully general (delete?) and move it out of this file
type FieldSelector []string

func (f FieldSelector) Select(record map[string]interface{}) (interface{}, error) {
	for i, nested := range f {
		recordInterface, ok := record[nested]
		if !ok {
			return nil, fmt.Errorf("field '%s' does not exist on record", nested)
		}

		if i == len(f)-1 {
			return recordInterface, nil
		}

		record, ok = recordInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot continue traversing record because field '%s' is not a map", nested)
		}
	}

	return nil, fmt.Errorf("should never get here")
}

func (f FieldSelector) Delete(record map[string]interface{}) error {
	for i, nested := range f {
		recordInterface, ok := record[nested]
		if !ok {
			return fmt.Errorf("field '%s' does not exist on record", nested)
		}

		if i == len(f)-1 {
			delete(record, nested)
			return nil
		}

		record, ok = recordInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot continue traversing record because field '%s' is not a map", nested)
		}
	}

	return fmt.Errorf("should never get here")
}

// TimeParserConfig is the configuration of a time parser plugin.
type TimeParserConfig struct {
	base.ParserConfig `mapstructure:",squash" yaml:",inline"`

	Field        FieldSelector
	Layout       string
	KeepOriginal bool `yaml:"keep_original"`
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	parserPlugin, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Field == nil {
		return nil, fmt.Errorf("missing required field 'field'")
	}

	if c.Layout == "" {
		return nil, fmt.Errorf("missing required field 'layout'")
	}

	timeParserPlugin := &TimeParser{
		ParserPlugin: parserPlugin,

		field:        c.Field,
		layout:       c.Layout,
		keepOriginal: c.KeepOriginal,
	}

	return timeParserPlugin, nil
}

// TimeParser is a plugin that parses time from a field.
type TimeParser struct {
	base.ParserPlugin

	field        FieldSelector
	layout       string
	keepOriginal bool
}

// Consume will parse time and send the entry to the next plugin.
func (p *TimeParser) Consume(entry *entry.Entry) error {
	newEntry, err := p.parseTime(entry)
	if err != nil {
		// TODO allow continuing with best effort
		return err
	}

	return p.Output.Consume(newEntry)
}

// Parse time will parse time and create a new entry.
func (p *TimeParser) parseTime(entry *entry.Entry) (*entry.Entry, error) {
	message, err := p.field.Select(entry.Record)
	if err != nil {
		return nil, fmt.Errorf("failed to select field: %s", err)
	}

	// TODO support bytes?
	messageString, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("field '%s' can not be parsed with regex because it is of type %T", p.field, message)
	}

	time, err := time.Parse(p.layout, messageString)
	if err != nil {
		return nil, fmt.Errorf("parsing time: %s", err)
	}

	if !p.keepOriginal {
		err := p.field.Delete(entry.Record)
		if err != nil {
			return nil, err
		}
	}

	entry.Timestamp = time

	return entry, nil
}
