package entry

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type FieldSelector interface {
	Get(Record) (interface{}, bool)
	Set(*Record, interface{})
	// Merge(Record, map[string]interface{})
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
