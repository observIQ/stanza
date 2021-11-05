package xml

import (
	"bytes"
	"encoding/xml"
)

// Element represents an XML element
type Element struct {
	Tag        string
	Content    string
	Attributes map[string]string
	Children   []*Element
	Parent     *Element
}

// convertToMap converts an element to a map
func convertToMap(element *Element) map[string]interface{} {
	results := map[string]interface{}{}
	results["tag"] = element.Tag

	if element.Content != "" {
		results["content"] = element.Content
	}

	if len(element.Attributes) > 0 {
		results["attributes"] = element.Attributes
	}

	if len(element.Children) > 0 {
		results["children"] = convertToMaps(element.Children)
	}

	return results
}

// convertToMaps converts a slice of elements to a slice of maps
func convertToMaps(elements []*Element) []map[string]interface{} {
	results := []map[string]interface{}{}
	for _, e := range elements {
		results = append(results, convertToMap(e))
	}

	return results
}

// newElement creates a new element for the given xml start element
func newElement(element xml.StartElement) *Element {
	return &Element{
		Tag:        element.Name.Local,
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
