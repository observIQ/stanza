package entry

import (
	"fmt"
	"time"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	// TODO consider using a more allocation-efficient representation
	Record Record `json:"record"`
}

// NewEntry will create a new log entry.
func NewEntry() *Entry {
	return &Entry{
		Timestamp: time.Now(),
		Record:    map[string]interface{}{},
	}
}

// Record
type Record interface{}

func (entry *Entry) Get(selector FieldSelector) (interface{}, bool) {
	return selector.Get(entry.Record)
}

func (entry *Entry) Set(selector FieldSelector, val interface{}) {
	selector.Set(&entry.Record, val)
}

func (entry *Entry) Delete(selector FieldSelector) (interface{}, bool) {
	return selector.Delete(&entry.Record)
}

func (entry *Entry) Read(selector FieldSelector, dest interface{}) error {
	val, ok := entry.Get(selector)
	if !ok {
		return fmt.Errorf("Field does not exist")
	}

	switch dest := dest.(type) {
	case *string:
		if str, ok := val.(string); ok {
			*dest = str
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to string", selector, val)
		}
	case *map[string]interface{}:
		if m, ok := val.(map[string]interface{}); ok {
			*dest = m
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to map[string]interface{}", selector, val)
		}
	case *map[string]string:
		switch m := val.(type) {
		case map[string]interface{}:
			newDest := make(map[string]string)
			for k, v := range m {
				if vStr, ok := v.(string); ok {
					newDest[k] = vStr
				} else {
					return fmt.Errorf("can not cast map members '%s' of type '%s' to string", k, v)
				}
			}
			*dest = newDest
		case map[interface{}]interface{}:
			newDest := make(map[string]string)
			for k, v := range m {
				kStr, ok := k.(string)
				if !ok {
					return fmt.Errorf("can not cast map key of type '%T' to string", k)
				}
				vStr, ok := v.(string)
				if !ok {
					return fmt.Errorf("can not cast map value of type '%T' to string", v)
				}
				newDest[kStr] = vStr
			}
			*dest = newDest
		}

	case *interface{}:
		*dest = val
	default:
		return fmt.Errorf("can not read to unsupported type '%T'", dest)
	}

	return nil

}
