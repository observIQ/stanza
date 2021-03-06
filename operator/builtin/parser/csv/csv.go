package csv

import (
	"context"
	csvparser "encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("csv_parser", func() operator.Builder { return NewCSVParserConfig("") })
}

// NewCSVParserConfig creates a new csv parser config with default values
func NewCSVParserConfig(operatorID string) *CSVParserConfig {
	return &CSVParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "csv_parser"),
	}
}

// CSVParserConfig is the configuration of a csv parser operator.
type CSVParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Header         string `json:"header" yaml:"header"`
	FieldDelimiter string `json:"delimiter,omitempty" yaml:"delimiter,omitempty"`
}

// Build will build a csv parser operator.
func (c CSVParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Header == "" {
		return nil, fmt.Errorf("Missing required field 'header'")
	}

	if c.FieldDelimiter == "" {
		c.FieldDelimiter = ","
	}

	if len([]rune(c.FieldDelimiter)) != 1 {
		return nil, fmt.Errorf("Invalid 'delimiter': '%s'", c.FieldDelimiter)
	}

	fieldDelimiter := []rune(c.FieldDelimiter)[0]

	if !strings.Contains(c.Header, c.FieldDelimiter) {
		return nil, fmt.Errorf("missing field delimiter in header")
	}

	numFields := len(strings.Split(c.Header, c.FieldDelimiter))

	csvParser := &CSVParser{
		ParserOperator: parserOperator,
		header:         c.Header,
		fieldDelimiter: fieldDelimiter,
		numFields:      numFields,
	}

	return []operator.Operator{csvParser}, nil
}

// CSVParser is an operator that parses csv in an entry.
type CSVParser struct {
	helper.ParserOperator
	header         string
	fieldDelimiter rune
	numFields      int
}

// Process will parse an entry for csv.
func (r *CSVParser) Process(ctx context.Context, entry *entry.Entry) error {
	return r.ParserOperator.ProcessWith(ctx, entry, r.parse)
}

// parse will parse a value using the supplied csv header.
func (r *CSVParser) parse(value interface{}) (interface{}, error) {
	var csvLine string
	switch value.(type) {
	case string:
		csvLine += value.(string)
	case []byte:
		csvLine += string(value.([]byte))
	default:
		return nil, fmt.Errorf("type '%T' cannot be parsed as csv", value)
	}

	delimiterStr := string([]rune{r.fieldDelimiter})

	reader := csvparser.NewReader(strings.NewReader(csvLine))
	reader.Comma = r.fieldDelimiter
	reader.FieldsPerRecord = r.numFields
	parsedValues := make(map[string]interface{})

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		for i, key := range strings.Split(r.header, delimiterStr) {
			parsedValues[key] = record[i]
		}
	}

	return parsedValues, nil
}
