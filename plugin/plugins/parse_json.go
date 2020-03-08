package plugins

import (
	"fmt"

	e "github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/json-iterator/go"
)

func init() {
	pg.RegisterConfig("json_parser", &JSONParserConfig{})
}

type JSONParserConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field            string
	DestinationField string
}

func (c JSONParserConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	if c.Field == "" {
		return nil, fmt.Errorf("missing required field 'field'")
	}

	plugin := &JSONParser{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,

		field:            c.Field,
		destinationField: c.DestinationField,
		json:             jsoniter.ConfigFastest,
	}

	return plugin, nil
}

type JSONParser struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	field            string
	destinationField string
	json             jsoniter.API
}

func (p *JSONParser) Input(entry *e.Entry) error {
	newEntry, err := p.processEntry(entry)
	if err != nil {
		// TODO option to allow
		return err
	}

	return p.Output(newEntry)
}

func (p *JSONParser) processEntry(entry *e.Entry) (*e.Entry, error) {
	message, ok := entry.Record[p.field]
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist on the record", p.field)
	}

	messageString, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("field '%s' can not be parsed as JSON because it is of type %T", p.field, message)
	}

	var parsedMessage map[string]interface{}
	err := p.json.UnmarshalFromString(messageString, &parsedMessage)
	if err != nil {
		return nil, fmt.Errorf("parse field %s as JSON: %w", p.field, err)
	}

	if p.destinationField == "" {
		entry.Record[p.field] = parsedMessage
	} else {
		entry.Record[p.destinationField] = parsedMessage
	}

	return entry, nil
}
