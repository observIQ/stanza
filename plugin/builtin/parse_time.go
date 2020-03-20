package builtin

import (
	"fmt"
	"time"

	e "github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("parse_time", &TimeParserConfig{})
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

type TimeParserConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field        FieldSelector
	Layout       string
	KeepOriginal bool `yaml:"keep_original"`
}

func (c TimeParserConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	if c.Field == nil {
		return nil, fmt.Errorf("missing required field 'field'")
	}

	if c.Layout == "" {
		return nil, fmt.Errorf("missing required field 'layout'")
	}

	plugin := &TimeParser{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,

		field:        c.Field,
		layout:       c.Layout,
		keepOriginal: c.KeepOriginal,
	}

	return plugin, nil
}

type TimeParser struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	field        FieldSelector
	layout       string
	keepOriginal bool
}

func (p *TimeParser) Input(entry *e.Entry) error {
	newEntry, err := p.processEntry(entry)
	if err != nil {
		// TODO allow continuing with best effort
		return err
	}

	return p.Output(newEntry)
}

func (p *TimeParser) processEntry(entry *e.Entry) (*e.Entry, error) {
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

// TODO write tests
