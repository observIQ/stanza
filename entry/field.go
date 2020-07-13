package entry

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Field represents a potential field on an entry.
// It is used to get, set, and delete values at this field.
// It is deserialized from JSON dot notation.
type Field struct {
	FieldInterface
}

// FieldInterface is a field on an entry.
type FieldInterface interface {
	Get(*Entry) (interface{}, bool)
	Set(entry *Entry, value interface{}) error
	Delete(entry *Entry) (interface{}, bool)
	String() string
}

// UnmarshalJSON will unmarshal a field from JSON
func (f *Field) UnmarshalJSON(raw []byte) error {
	var s string
	err := json.Unmarshal(raw, &s)
	if err != nil {
		return err
	}
	*f, err = fieldFromString(s)
	return err
}

// UnmarshalYAML will unmarshal a field from YAML
func (f *Field) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	*f, err = fieldFromString(s)
	return err
}

func fieldFromString(s string) (Field, error) {
	split := strings.Split(s, ".")

	switch split[0] {
	case "$labels":
		if len(split) != 2 {
			return Field{}, fmt.Errorf("labels cannot be nested")
		}
		return Field{LabelField{split[1]}}, nil
	case "$record", "$":
		return Field{RecordField{split[1:]}}, nil
	default:
		return Field{RecordField{split}}, nil
	}
}

// MarshalJSON will marshal a field into JSON
func (f Field) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", f.String())), nil
}

// MarshalYAML will marshal a field into YAML
func (f Field) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}
