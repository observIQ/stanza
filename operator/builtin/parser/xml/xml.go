package xml

import (
	"bytes"
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

// NewXMLParserConfig creates a new JSON parser config with default values
func NewXMLParserConfig(operatorID string) *XMLParserConfig {
	return &XMLParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "xml_parser"),
	}
}

// XMLParserConfig is the configuration of a JSON parser operator.
type XMLParserConfig struct {
	helper.ParserConfig `yaml:",inline"`
}

// Build will build a JSON parser operator.
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

// XMLParser is an operator that parses JSON.
type XMLParser struct {
	helper.ParserOperator
}

// Process will parse an entry for JSON.
func (x *XMLParser) Process(ctx context.Context, entry *entry.Entry) error {
	return x.ParserOperator.ProcessWith(ctx, entry, Parse)
}

// Document is the root level of an XML document
type Document struct {
	Nodes []*Node `json:"nodes"`
}

// Node represents an XML element
type Node struct {
	Type       string            `json:"type"`
	Value      string            `json:"value,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Children   []*Node           `json:"children,omitempty"`
	Parent     *Node             `json:"-"`
}

// Parse will parse an xml document
func Parse(value interface{}) (interface{}, error) {
	strValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("Value is not a string")
	}

	reader := strings.NewReader(strValue)
	decoder := xml.NewDecoder(reader)
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get decoder token: %w", err)
	}

	document := &Document{}
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
				document.Nodes = append(document.Nodes, child)
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
			return nil, fmt.Errorf("failed to get next token: %w", err)
		}
	}

	return document, nil
}

// newNode creates a new node for the given element
func newNode(element xml.StartElement) *Node {
	return &Node{
		Type:       element.Name.Local,
		Attributes: getAttributes(element),
	}
}

// getAttributes returns the attributes of the given element
func getAttributes(element xml.StartElement) map[string]string {
	if len(element.Attr) == 0 {
		return nil
	}

	attributes := map[string]string{}
	for _, attr := range element.Attr {
		key := attr.Name.Local
		attributes[key] = attr.Value
	}

	return attributes
}

// getValue returns value of the given char data
func getValue(data xml.CharData) string {
	return string(bytes.TrimSpace(data))
}

// getNumOfElements returns the num of XML elements currently stored in an array of Nodes
func getNumOfElements(nodes []*Node) int {
	sum := 0

	if len(nodes) == 0 {
		return 0
	}

	for _, i := range nodes {
		sum += 1 + getNumOfElements(i.Children)
	}

	return sum
}
