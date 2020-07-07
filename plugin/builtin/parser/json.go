package parser

import (
	"context"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/bplogagent/entry"
	"github.com/observiq/bplogagent/plugin"
	"github.com/observiq/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("json_parser", &JSONParserConfig{})
}

// JSONParserConfig is the configuration of a JSON parser plugin.
type JSONParserConfig struct {
	helper.ParserConfig `yaml:",inline"`
}

// Build will build a JSON parser plugin.
func (c JSONParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	parserPlugin, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	jsonParser := &JSONParser{
		ParserPlugin: parserPlugin,
		json:         jsoniter.ConfigFastest,
	}

	return jsonParser, nil
}

// JSONParser is a plugin that parses JSON.
type JSONParser struct {
	helper.ParserPlugin
	json jsoniter.API
}

// Process will parse an entry for JSON.
func (j *JSONParser) Process(ctx context.Context, entry *entry.Entry) error {
	return j.ParserPlugin.ProcessWith(ctx, entry, j.parse)
}

// parse will parse a value as JSON.
func (j *JSONParser) parse(value interface{}) (interface{}, error) {
	var parsedValue map[string]interface{}
	switch m := value.(type) {
	case string:
		err := j.json.UnmarshalFromString(m, &parsedValue)
		if err != nil {
			return nil, err
		}
	case []byte:
		err := j.json.Unmarshal(m, &parsedValue)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("type %T cannot be parsed as JSON", value)
	}
	return parsedValue, nil
}
