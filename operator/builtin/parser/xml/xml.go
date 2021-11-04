package xml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
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
}

// Build will build an XML parser operator.
func (c XMLParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	xmlParser := &XMLParser{
		ParserOperator: parserOperator,
	}

	return []operator.Operator{xmlParser}, nil
}

// XMLParser is an operator that parses XML.
type XMLParser struct {
	helper.ParserOperator
}

// Process will parse an entry for XML.
func (x *XMLParser) Process(ctx context.Context, entry *entry.Entry) error {
	return x.ParserOperator.ProcessWith(ctx, entry, parse)
}

// parse will parse an xml value
func parse(value interface{}) (interface{}, error) {
	strValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value is not a string")
	}

	reader := strings.NewReader(strValue)
	decoder := xml.NewDecoder(reader)
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to decode as xml: %w", err)
	}

	nodes := []*Node{}
	var parent *Node
	var child *Node

	for token != nil {
		switch token := token.(type) {
		case xml.StartElement:
			parent = child
			child = newNode(token)
			child.Parent = parent

			if parent != nil {
				parent.Children = append(parent.Children, child)
			} else {
				nodes = append(nodes, child)
			}
		case xml.EndElement:
			child = parent
			if parent != nil {
				parent = parent.Parent
			}
		case xml.CharData:
			if child != nil {
				child.Value = getValue(token)
			}
		}

		token, err = decoder.Token()
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to get next xml token: %w", err)
		}
	}

	switch len(nodes) {
	case 0:
		return nil, fmt.Errorf("no xml nodes found")
	case 1:
		return convertToMap(nodes[0]), nil
	default:
		return convertToMaps(nodes), nil
	}
}
