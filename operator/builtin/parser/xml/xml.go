package xml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("xml_parser", func() operator.Builder { return NewXMLParserConfig("") })
}

// NewXMLParserConfig creates a new XML parser config with default values
func NewXMLParserConfig(operatorID string) *XMLParserConfig {
	return &XMLParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "xml_parser"),
	}
}

// XMLParserConfig is the configuration of an XML parser operator.
type XMLParserConfig struct {
	helper.ParserConfig `yaml:",inline"`
	Strict              *bool `yaml:"strict,omitempty"`
}

// Build will build an XML parser operator.
func (c XMLParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	strict := true
	if c.Strict != nil {
		strict = *c.Strict
	}

	xmlParser := &XMLParser{
		ParserOperator: parserOperator,
		strict:         strict,
	}

	return []operator.Operator{xmlParser}, nil
}

// XMLParser is an operator that parses XML.
type XMLParser struct {
	helper.ParserOperator
	strict bool
}

// Process will parse an entry for XML.
func (x *XMLParser) Process(ctx context.Context, entry *entry.Entry) error {
	return x.ParserOperator.ProcessWith(ctx, entry, x.parse)
}

// parse will parse an xml value
func (x *XMLParser) parse(value interface{}) (interface{}, error) {
	strValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value passed to parser is not a string")
	}

	reader := strings.NewReader(strValue)
	decoder := xml.NewDecoder(reader)
	decoder.Strict = x.strict
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to decode as xml: %w", err)
	}

	elements := []*Element{}
	var parent *Element
	var current *Element

	for token != nil {
		switch token := token.(type) {
		case xml.StartElement:
			parent = current
			current = newElement(token)
			current.Parent = parent

			if parent != nil {
				parent.Children = append(parent.Children, current)
			} else {
				elements = append(elements, current)
			}
		case xml.EndElement:
			current = parent
			if parent != nil {
				parent = parent.Parent
			}
		case xml.CharData:
			if current != nil {
				current.Content = getValue(token)
			}
		}

		token, err = decoder.Token()
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to get next xml token: %w", err)
		}
	}

	switch len(elements) {
	case 0:
		return nil, fmt.Errorf("no xml elements found")
	case 1:
		return convertToMap(elements[0]), nil
	default:
		return convertToMaps(elements), nil
	}
}
