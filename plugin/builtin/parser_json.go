package builtin

import (
	"context"
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
	helper.BasicPluginConfig `yaml:",inline"`
	helper.BasicParserConfig `yaml:",inline"`
}

// Build will build a JSON parser plugin.
func (c JSONParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicParser, err := c.BasicParserConfig.Build(basicPlugin.SugaredLogger)
	if err != nil {
		return nil, err
	}

	jsonParser := &JSONParser{
		BasicPlugin: basicPlugin,
		BasicParser: basicParser,
		json:        jsoniter.ConfigFastest,
	}

	return jsonParser, nil
}

// JSONParser is a plugin that parses JSON.
type JSONParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicParser
	json jsoniter.API
}

// Process will parse an entry for JSON.
func (j *JSONParser) Process(ctx context.Context, entry *entry.Entry) error {
	return j.BasicParser.ProcessWith(ctx, entry, j.parse)
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
