package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("json_parser", &JSONParserConfig{})
}

// JSONParserConfig is the configuration of a JSON parser plugin.
type JSONParserConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field            *entry.FieldSelector
	DestinationField *entry.FieldSelector
}

// Build will build a JSON parser plugin.
func (c JSONParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Field == nil {
		var fs entry.FieldSelector = entry.SingleFieldSelector([]string{})
		c.Field = &fs
	}

	if c.DestinationField == nil {
		c.DestinationField = c.Field
	}

	plugin := &JSONParser{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,

		field:            *c.Field,
		destinationField: *c.DestinationField,
		json:             jsoniter.ConfigFastest,
	}

	return plugin, nil
}

// JSONParser is a plugin that parses JSON.
type JSONParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	field            entry.FieldSelector
	destinationField entry.FieldSelector
	json             jsoniter.API
}

// Process will parse an entry field as JSON.
func (p *JSONParser) Process(entry *entry.Entry) error {
	newEntry, err := p.parse(entry)
	if err != nil {
		p.Warnw("Failed to parse message", zap.Error(err), "message", entry)
		return nil
	}

	return p.Output.Process(newEntry)
}

// parse will parse an entry.
func (p *JSONParser) parse(entry *entry.Entry) (*entry.Entry, error) {
	message, ok := entry.Get(p.field)
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist on the record", p.field)
	}

	var parsedMessage map[string]interface{}
	switch m := message.(type) {
	case string:
		err := p.json.UnmarshalFromString(m, &parsedMessage)
		if err != nil {
			return nil, fmt.Errorf("parse field %s as JSON: %w", p.field, err)
		}
	case []byte:
		err := p.json.Unmarshal(m, &parsedMessage)
		if err != nil {
			return nil, fmt.Errorf("parse field %s as JSON: %w", p.field, err)
		}
	default:
		return nil, fmt.Errorf("cannot parse field of type %T as JSON", message)
	}

	entry.Set(p.destinationField, parsedMessage)
	return entry, nil
}
