package entry

import (
	"fmt"
	"strings"
)

// LabelField is the path to an entry label
type LabelField struct {
	key string
}

// Get will return the label value and a boolean indicating if it exists
func (l LabelField) Get(entry *Entry) (interface{}, bool) {
	if entry.Attributes == nil {
		return "", false
	}
	val, ok := entry.Attributes[l.key]
	return val, ok
}

// Set will set the label value on an entry
func (l LabelField) Set(entry *Entry, val interface{}) error {
	if entry.Attributes == nil {
		entry.Attributes = make(map[string]string, 1)
	}

	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("cannot set a label to a non-string value")
	}
	entry.Attributes[l.key] = str
	return nil
}

// Delete will delete a label from an entry
func (l LabelField) Delete(entry *Entry) (interface{}, bool) {
	if entry.Attributes == nil {
		return "", false
	}

	val, ok := entry.Attributes[l.key]
	delete(entry.Attributes, l.key)
	return val, ok
}

func (l LabelField) String() string {
	if strings.Contains(l.key, ".") {
		return fmt.Sprintf(`$attributes['%s']`, l.key)
	}
	return "$attributes." + l.key
}

// NewLabelField will creat a new label field from a key
func NewLabelField(key string) Field {
	return Field{LabelField{key}}
}
