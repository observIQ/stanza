package xml

import (
	"bytes"
	"encoding/xml"
)

// Node represents an XML element
type Node struct {
	Type       string
	Value      string
	Attributes map[string]string
	Children   []*Node
	Parent     *Node
}

// convertToMap converts a node to a map
func convertToMap(node *Node) map[string]interface{} {
	results := map[string]interface{}{}
	results["type"] = node.Type

	if node.Value != "" {
		results["value"] = node.Value
	}

	if len(node.Attributes) > 0 {
		results["attributes"] = node.Attributes
	}

	if len(node.Children) > 0 {
		results["children"] = convertToMaps(node.Children)
	}

	return results
}

// convertToMaps converts a slice of nodes to a slice of maps
func convertToMaps(nodes []*Node) []map[string]interface{} {
	results := []map[string]interface{}{}
	for _, node := range nodes {
		results = append(results, convertToMap(node))
	}

	return results
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
