package entry

import (
	"encoding/json"
	"fmt"
)

const (
	attributesPrefix = "$attributes"
	resourcePrefix   = "$resource"
	recordPrefix     = "$record"
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
	case attributesPrefix:
		if len(split) != 2 {
			return Field{}, fmt.Errorf("attributes cannot be nested")
		}
		return Field{LabelField{split[1]}}, nil
	case resourcePrefix:
		if len(split) != 2 {
			return Field{}, fmt.Errorf("resource fields cannot be nested")
		}
		return Field{ResourceField{split[1]}}, nil
	case recordPrefix, "$":
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
	// Begin is the beginning state of a field split
	Begin splitState = iota
	// InBracket is the state of a field split inside a bracket
	InBracket
	// InQuote is the state of a field split inside a quote
	InQuote
	// OutQuote is the state of a field split outside a quote
	OutQuote
	// OutBracket is the state of a field split outside a bracket
	OutBracket
	// InUnbracketedToken is the state field split on any token outside brackets
	InUnbracketedToken
)

func splitField(s string) ([]string, error) {
	fields := make([]string, 0, 1)

	state := Begin
	var quoteChar rune
	var tokenStart int

	for i, c := range s {
		switch state {
		case Begin:
			if c == '[' {
				state = InBracket
				continue
			}
			tokenStart = i
			state = InUnbracketedToken
		case InBracket:
			if !(c == '\'' || c == '"') {
				return nil, fmt.Errorf("strings in brackets must be surrounded by quotes")
			}
			state = InQuote
			quoteChar = c
			tokenStart = i + 1
		case InQuote:
			if c == quoteChar {
				fields = append(fields, s[tokenStart:i])
				state = OutQuote
			}
		case OutQuote:
			if c != ']' {
				return nil, fmt.Errorf("found characters between closed quote and closing bracket")
			}
			state = OutBracket
		case OutBracket:
			switch c {
			case '.':
				state = InUnbracketedToken
				tokenStart = i + 1
			case '[':
				state = InBracket
			default:
				return nil, fmt.Errorf("bracketed access must be followed by a dot or another bracketed access")
			}
		case InUnbracketedToken:
			if c == '.' {
				fields = append(fields, s[tokenStart:i])
				tokenStart = i + 1
			} else if c == '[' {
				fields = append(fields, s[tokenStart:i])
				state = InBracket
			}
		}
	}

	switch state {
	case InBracket, OutQuote:
		return nil, fmt.Errorf("found unclosed left bracket")
	case InQuote:
		if quoteChar == '"' {
			return nil, fmt.Errorf("found unclosed double quote")
		}
		return nil, fmt.Errorf("found unclosed single quote")
	case InUnbracketedToken:
		fields = append(fields, s[tokenStart:])
	}

	return fields, nil
}
