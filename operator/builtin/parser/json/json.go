package json

import (
	"context"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("json_parser", func() operator.Builder { return NewJSONParserConfig("") })
}

// NewJSONParserConfig creates a new JSON parser config with default values
func NewJSONParserConfig(operatorID string) *JSONParserConfig {
	return &JSONParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "json_parser"),
	}
}

// JSONParserConfig is the configuration of a JSON parser operator.
type JSONParserConfig struct {
	helper.ParserConfig `yaml:",inline"`
}

// Build will build a JSON parser operator.
func (c JSONParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	jsonParser := &JSONParser{
		ParserOperator: parserOperator,
		json:           jsoniter.ConfigFastest,
	}

	return []operator.Operator{jsonParser}, nil
}

// JSONParser is an operator that parses JSON.
type JSONParser struct {
	helper.ParserOperator
	json jsoniter.API
}

// Process will parse an entry for JSON.
func (j *JSONParser) Process(ctx context.Context, entry *entry.Entry) error {
	return j.ParserOperator.ProcessWith(ctx, entry, j.parse)
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
