package entry

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type FieldSelector interface {
	Get(Record) (interface{}, bool)
	Set(*Record, interface{})
	Delete(*Record) (interface{}, bool)
	// Merge(Record, map[string]interface{})
}

func NewSingleFieldSelector(fields ...string) FieldSelector {
	fs := SingleFieldSelector(fields)
	return &fs
}

// TODO support arrays?
type SingleFieldSelector []string

func (s SingleFieldSelector) Get(record Record) (interface{}, bool) {
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
func (s SingleFieldSelector) Set(record *Record, val interface{}) {
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
func (s SingleFieldSelector) Delete(record *Record) (interface{}, bool) {
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
		return SingleFieldSelector([]string{data.(string)}), nil
	case reflect.TypeOf([]string{}):
		return SingleFieldSelector(data.([]string)), nil
	default:
		return nil, fmt.Errorf("cannot unmarshal an entry.FieldSelector from type %s", f)
	}
}
