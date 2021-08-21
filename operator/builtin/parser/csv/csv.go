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

	Header          string `json:"header" yaml:"header"`
	HeaderLabel     string `json:"header_label" yaml:"header_label"`
	HeaderDelimiter string `json:"header_delimiter,omitempty" yaml:"header_delimiter,omitempty"`
	FieldDelimiter  string `json:"delimiter,omitempty" yaml:"delimiter,omitempty"`
}

// Build will build a csv parser operator.
func (c CSVParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Header == "" && c.HeaderLabel == "" {
		return nil, fmt.Errorf("missing required field 'header' or 'header_label'")
	}

	if c.Header != "" && c.HeaderLabel != "" {
		return nil, fmt.Errorf("only one header parameter can be set: 'header' or 'header_label'")
	}

	// configure dynamic header
	dynamic := false
	if c.HeaderLabel != "" {
		dynamic = true
	}

	if c.FieldDelimiter == "" {
		c.FieldDelimiter = ","
	}

	if len([]rune(c.FieldDelimiter)) != 1 {
		return nil, fmt.Errorf("Invalid 'delimiter': '%s'", c.FieldDelimiter)
	}

	fieldDelimiter := []rune(c.FieldDelimiter)[0]

	if c.HeaderDelimiter == "" {
		c.HeaderDelimiter = c.FieldDelimiter
	}

	headerDelimiter := []rune(c.HeaderDelimiter)[0]

	if !dynamic && !strings.Contains(c.Header, c.HeaderDelimiter) {
		return nil, fmt.Errorf("missing header delimiter in header")
	}

	// encoding/csv defaults to 0 (auto detection), however, if number of fields
	// is known, set it here to avoid checking it in the parse function on every entry
	numFields := 0
	if !dynamic {
		numFields = len(strings.Split(c.Header, c.HeaderDelimiter))
	}

	csvParser := &CSVParser{
		ParserOperator:  parserOperator,
		header:          c.Header,
		headerLabel:     c.HeaderLabel,
		dynamicHeader:   dynamic,
		headerDelimiter: headerDelimiter,
		fieldDelimiter:  fieldDelimiter,
		numFields:       numFields,

		// initial parse function, overwritten when dynamic headers are enabled
		parse: generateParseFunc(c.Header, headerDelimiter, fieldDelimiter, numFields),
	}

	return []operator.Operator{csvParser}, nil
}

// CSVParser is an operator that parses csv in an entry.
type CSVParser struct {
	helper.ParserOperator
	header          string
	headerLabel     string
	dynamicHeader   bool
	headerDelimiter rune
	fieldDelimiter  rune
	numFields       int
	parse           ParseFunc
}

// Process will parse an entry for csv.
func (r *CSVParser) Process(ctx context.Context, e *entry.Entry) error {
	if r.dynamicHeader {
		h, ok := e.Labels[r.headerLabel]
		if !ok {
			// TODO: returned error is not logged, so log it here
			err := fmt.Errorf("failed to read dynamic header label %s", r.headerLabel)
			r.Error(err)
			return err
		}
		r.parse = generateParseFunc(h, r.headerDelimiter, r.fieldDelimiter, r.numFields)
	}
	return r.ParserOperator.ProcessWith(ctx, e, r.parse)
}

type ParseFunc func(interface{}) (interface{}, error)

// generateParseFunc returns a parse function for a given header, allowing
// each entry to have a potentially unique set of fields when using dynamic
// field names retrieved from an entry's label
func generateParseFunc(header string, headerDelimiter, fieldDelimiter rune, numFields int) ParseFunc {
	return func(value interface{}) (interface{}, error) {
		var csvLine string
		switch t := value.(type) {
		case string:
			csvLine += t
		case []byte:
			csvLine += string(t)
		default:
			return nil, fmt.Errorf("type '%T' cannot be parsed as csv", value)
		}

		reader := csvparser.NewReader(strings.NewReader(csvLine))
		reader.Comma = fieldDelimiter
		reader.FieldsPerRecord = numFields
		parsedValues := make(map[string]interface{})

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			headerFields := strings.Split(header, string([]rune{headerDelimiter}))

			// When numFields is less than 1 (due to dynamic header detection) we need to check if
			// the correct number of fields exists before looping. When numFields is greater than 0,
			// encoding/csv will return its own error when this mismatch occurs.
			if numFields < 1 {
				if len(headerFields) != len(record) {
					return nil, fmt.Errorf("number of fields does not match number of fields defined by the header '%s'", headerFields)
				}
			}

			for i, key := range headerFields {
				parsedValues[key] = record[i]
			}
		}

		return parsedValues, nil
	}
}
