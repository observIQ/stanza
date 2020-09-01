package entry

import (
	"encoding/json"
	"fmt"
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
	split, err := splitField(s)
	if err != nil {
		return Field{}, fmt.Errorf("splitting field: %s", err)
	}

	switch split[0] {
	case "$labels":
		if len(split) != 2 {
			return Field{}, fmt.Errorf("labels cannot be nested")
		}
		return Field{LabelField{split[1]}}, nil
	case "$resource":
		if len(split) != 2 {
			return Field{}, fmt.Errorf("resource fields cannot be nested")
		}
		return Field{ResourceField{split[1]}}, nil
	case "$record", "$":
		return Field{RecordField{split[1:]}}, nil
	default:
		return Field{RecordField{split}}, nil
	}
}

// MarshalJSON will marshal a field into JSON
func (f Field) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, f.String())), nil
}

// MarshalYAML will marshal a field into YAML
func (f Field) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}

type splitState uint

const (
	// BEGIN is the beginning state of a field split
	BEGIN splitState = iota
	// IN_BRACKET is the state of a field split inside a bracket
	IN_BRACKET
	// IN_QUOTE is the state of a field split inside a quote
	IN_QUOTE
	// OUT_QUOTE is the state of a field split outside a quote
	OUT_QUOTE
	// OUT_BRACKET is the state of a field split outside a bracket
	OUT_BRACKET
	// IN_UNBRACKETED_TOKEN is the state field split on any token outside brackets
	IN_UNBRACKETED_TOKEN
)

func splitField(s string) ([]string, error) {
	fields := make([]string, 0, 1)

	state := BEGIN
	var quoteChar rune
	var tokenStart int

	for i, c := range s {
		switch state {
		case BEGIN:
			if c == '[' {
				state = IN_BRACKET
				continue
			}
			tokenStart = i
			state = IN_UNBRACKETED_TOKEN
		case IN_BRACKET:
			if !(c == '\'' || c == '"') {
				return nil, fmt.Errorf("strings in brackets must be surrounded by quotes")
			}
			state = IN_QUOTE
			quoteChar = c
			tokenStart = i + 1
		case IN_QUOTE:
			if c == quoteChar {
				fields = append(fields, s[tokenStart:i])
				state = OUT_QUOTE
			}
		case OUT_QUOTE:
			if c != ']' {
				return nil, fmt.Errorf("found characters between closed quote and closing bracket")
			}
			state = OUT_BRACKET
		case OUT_BRACKET:
			if c == '.' {
				state = IN_UNBRACKETED_TOKEN
				tokenStart = i + 1
			} else if c == '[' {
				state = IN_BRACKET
			} else {
				return nil, fmt.Errorf("bracketed access must be followed by a dot or another bracketed access")
			}
		case IN_UNBRACKETED_TOKEN:
			if c == '.' {
				fields = append(fields, s[tokenStart:i])
				tokenStart = i + 1
			} else if c == '[' {
				fields = append(fields, s[tokenStart:i])
				state = IN_BRACKET
			}
		}
	}

	switch state {
	case IN_BRACKET, OUT_QUOTE:
		return nil, fmt.Errorf("found unclosed left bracket")
	case IN_QUOTE:
		if quoteChar == '"' {
			return nil, fmt.Errorf("found unclosed double quote")
		}
		return nil, fmt.Errorf("found unclosed single quote")
	case IN_UNBRACKETED_TOKEN:
		fields = append(fields, s[tokenStart:])
	}

	return fields, nil
}
