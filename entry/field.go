package entry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/bluemedora/bplogagent/errors"
	"github.com/mitchellh/mapstructure"
)

// Field represents a potential field on an entry's record.
// It is used to get, set, and delete values at this field.
// It is deserialized from JSON dot notation.
type Field struct {
	Keys []string
}

// Parent returns the parent of the current field.
// In the case that the record field points to the root node, it is a no-op.
func (f Field) Parent() Field {
	if f.IsRoot() {
		return f
	}

	keys := f.Keys[0 : f.Length()-1]
	return NewField(keys...)
}

// Child returns a child of the current field using the given key.
func (f Field) Child(key string) Field {
	child := make([]string, f.Length(), f.Length()+1)
	copy(child, f.Keys)
	keys := append(child, key)
	return NewField(keys...)
}

// IsRoot returns a boolean indicating if this is a root level field.
func (f Field) IsRoot() bool {
	return f.Length() == 0
}

// String returns the string representation of this field.
func (f Field) String() string {
	return toJSONDot(f)
}

// Length returns the number of keys in the field.
func (f Field) Length() int {
	return len(f.Keys)
}

// Get will retreive a value from an entry's record using the field.
// It will return the value and whether the field existed.
func (f Field) Get(entry *Entry) (interface{}, bool) {
	var currentValue interface{} = entry.Record

	for _, key := range f.Keys {
		currentRecord, ok := currentValue.(map[string]interface{})
		if !ok {
			return nil, false
		}

		currentValue, ok = currentRecord[key]
		if !ok {
			return nil, false
		}
	}

	return currentValue, true
}

// Set will set a value on an entry's record using the field.
// It will overwrite intermediate values as necessary.
func (f Field) Set(entry *Entry, value interface{}) {
	if f.IsRoot() {
		entry.Record = value
		return
	}

	mapValue, isMapValue := value.(map[string]interface{})
	if isMapValue {
		f.Merge(entry, mapValue)
		return
	}

	currentMap, ok := entry.Record.(map[string]interface{})
	if !ok {
		currentMap = map[string]interface{}{}
		entry.Record = currentMap
	}

	for i, key := range f.Keys {
		if i == f.Length()-1 {
			currentMap[key] = value
			return
		}
		currentMap = f.getNestedMap(currentMap, key)
	}
}

// Merge will attempt to merge the contents of a map into an entry's record.
// It will overwrite any intermediate values as necessary.
func (f Field) Merge(entry *Entry, mapValues map[string]interface{}) {
	currentMap, ok := entry.Record.(map[string]interface{})
	if !ok {
		currentMap = map[string]interface{}{}
		entry.Record = currentMap
	}

	for _, key := range f.Keys {
		currentMap = f.getNestedMap(currentMap, key)
	}

	for key, value := range mapValues {
		currentMap[key] = value
	}
}

// Delete removes a value from an entry's record using the field.
// It will return the deleted value and whether the field existed.
func (f Field) Delete(entry *Entry) (interface{}, bool) {
	if f.IsRoot() {
		oldRecord := entry.Record
		entry.Record = nil
		return oldRecord, true
	}

	currentValue := entry.Record
	for i, key := range f.Keys {
		currentMap, ok := currentValue.(map[string]interface{})
		if !ok {
			return nil, false
		}

		currentValue, ok = currentMap[key]
		if !ok {
			return nil, false
		}

		if i == f.Length()-1 {
			delete(currentMap, key)
			return currentValue, true
		}
	}

	return nil, false
}

// getNestedMap will get a nested map assigned to a key.
// If the map does not exist, it will create and return it.
func (f Field) getNestedMap(currentMap map[string]interface{}, key string) map[string]interface{} {
	currentValue, ok := currentMap[key]
	if !ok {
		currentMap[key] = map[string]interface{}{}
	}

	nextMap, ok := currentValue.(map[string]interface{})
	if !ok {
		nextMap = map[string]interface{}{}
		currentMap[key] = nextMap
	}

	return nextMap
}

/****************
  Serialization
****************/

// UnmarshalJSON will attempt to unmarshal the field from JSON.
func (f *Field) UnmarshalJSON(raw []byte) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("the field is not a string: %s", err)
	}

	*f = fromJSONDot(value)
	return nil
}

// MarshalJSON will marshal the field for JSON.
func (f Field) MarshalJSON() ([]byte, error) {
	json := fmt.Sprintf(`"%s"`, toJSONDot(f))
	return []byte(json), nil
}

// UnmarshalYAML will attempt to unmarshal a field from YAML.
func (f *Field) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return fmt.Errorf("the field is not a string: %s", err)
	}

	*f = fromJSONDot(value)
	return nil
}

// MarshalYAML will marshal the field for YAML.
func (f Field) MarshalYAML() (interface{}, error) {
	return toJSONDot(f), nil
}

// fromJSONDot creates a field from JSON dot notation.
func fromJSONDot(value string) Field {
	keys := strings.Split(value, ".")

	if keys[0] == "$" {
		keys = keys[1:]
	}

	return NewField(keys...)
}

// toJSONDot returns the JSON dot notation for a field.
func toJSONDot(field Field) string {
	if field.IsRoot() {
		return "$"
	}

	return strings.Join(field.Keys, ".")
}

// NewField creates a new field from an ordered array of keys.
func NewField(keys ...string) Field {
	return Field{
		Keys: keys,
	}
}

// FieldDecoder is a custom decoder hook used by mapstructure to decode fields.
var FieldDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t.String() != "entry.Field" {
		return data, nil
	}

	if reflect.TypeOf(string("")) == f {
		return fromJSONDot(data.(string)), nil
	}

	return nil, errors.NewError(
		"failed to unmarshal field from type",
		"ensure that all fields are encoded as strings",
		"type", f.String())
}
