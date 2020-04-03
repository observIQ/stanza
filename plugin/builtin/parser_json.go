package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	plugin.Register("json_parser", &JSONParserConfig{})
}

// JSONParserConfig is the configuration of a JSON parser plugin.
type JSONParserConfig struct {
	helper.BasicIdentityConfig    `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field            string
	DestinationField string
}

// Build will build a JSON parser plugin.
func (c JSONParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicIdentity, err := c.BasicIdentityConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	plugin := &JSONParser{
		BasicIdentity:    basicIdentity,
		BasicTransformer: basicTransformer,

		field:            c.Field,
		destinationField: c.DestinationField,
		json:             jsoniter.ConfigFastest,
	}

	return plugin, nil
}

// JSONParser is a plugin that parses JSON.
type JSONParser struct {
	helper.BasicIdentity
	helper.BasicLifecycle
	helper.BasicTransformer

	field            string
	destinationField string
	json             jsoniter.API
}

// Process will parse an entry field as JSON.
func (p *JSONParser) Process(entry *entry.Entry) error {
	newEntry, err := p.parse(entry)
	if err != nil {
		// TODO option to allow
		return err
	}

	return p.Output.Process(newEntry)
}

// parse will parse an entry.
func (p *JSONParser) parse(entry *entry.Entry) (*entry.Entry, error) {
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
