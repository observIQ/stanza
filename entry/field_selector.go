package entry

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// TODO support arrays?
type FieldSelector []string

// Parent returns the parent field selector of the field selector.
// In the case that the field selector points to the root node,
// it is a no-op
func (s FieldSelector) Parent() FieldSelector {
	if len(s) == 0 {
		return s
	}

	return s[0 : len(s)-1]
}

// Child returns the selector for the child of the current field
// selector with the given key
func (s FieldSelector) Child(key string) FieldSelector {
	newSelector := make([]string, len(s), len(s)+1)
	copy(newSelector, s)
	return append(newSelector, key)
}

func (s FieldSelector) Get(record Record) (interface{}, bool) {
	var current interface{} = record
	for _, str := range s {
		mapNext, ok := current.(map[string]interface{})
		if !ok {
			// The current level is not a map,
			return nil, false
		}

		current, ok = mapNext[str]
		if !ok {
			// The current level's key does not exist
			return nil, false
		}
	}

	return current, true
}

// Set sets a value, overwriting any intermediate values as necessary
func (s FieldSelector) Set(record *Record, val interface{}) {
	if len(s) == 0 {
		*record = Record(val)
		return
	}

	var currentMap map[string]interface{}
	var ok bool
	currentMap, ok = (*record).(map[string]interface{})
	if !ok {
		currentMap = map[string]interface{}{}
		*record = currentMap
	}

	for i, str := range s {
		if i == len(s)-1 {
			currentMap[str] = val
			return
		}

		current, ok := currentMap[str]
		if !ok {
			current = map[string]interface{}{}
			currentMap[str] = current
		}

		next, ok := current.(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			currentMap[str] = next
		}

		currentMap = next
	}

	return
}

// Delete removes a field from a record. It returns the deleted field and
// whether the field existed
func (s FieldSelector) Delete(record *Record) (interface{}, bool) {
	if record == nil {
		return nil, false
	}

	if len(s) == 0 {
		old := *record
		*record = Record(map[string]interface{}{})
		return old, true
	}

	var currentMap map[string]interface{}
	var ok bool
	currentMap, ok = (*record).(map[string]interface{})
	if !ok {
		return nil, false
	}

	for i, str := range s {
		if i == len(s)-1 {
			old, ok := currentMap[str]
			if !ok {
				return nil, false
			}
			delete(currentMap, str)
			return old, true
		}

		current, ok := currentMap[str]
		if !ok {
			return nil, false
		}

		next, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}

		currentMap = next
	}

	return nil, false
}

var FieldSelectorDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t.String() != "entry.FieldSelector" {
		return data, nil
	}

	switch f {
	case reflect.TypeOf(string("")):
		return FieldSelector([]string{data.(string)}), nil
	case reflect.TypeOf([]string{}):
		return FieldSelector(data.([]string)), nil
	case reflect.TypeOf([]interface{}{}):
		newSlice := make([]string, 0, len(data.([]interface{})))

		for _, val := range data.([]interface{}) {
			strVar, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("cannot use type '%T' as part of a field selector", val)
			}
			newSlice = append(newSlice, strVar)
		}
		return FieldSelector(newSlice), nil
	default:
		return nil, fmt.Errorf("cannot unmarshal an entry.FieldSelector from type %s", f)
	}
}
