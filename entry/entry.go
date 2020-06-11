package entry

import (
	"fmt"
	"time"
)

// Entry is a flexible representation of log data associated with a timestamp.
type Entry struct {
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
	Tags      []string          `json:"tags,omitempty"      yaml:"tags,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"    yaml:"labels,omitempty"`
	Record    interface{}       `json:"record"    yaml:"record"`
}

// New will create a new log entry with current timestamp and an empty record.
func New() *Entry {
	return &Entry{
		Timestamp: time.Now(),
		Record:    map[string]interface{}{},
		Tags:      make([]string, 0),
		Labels:    make(map[string]string, 0),
	}
}

func (entry *Entry) Get(field Field) (interface{}, bool) {
	return field.Get(entry)
}

func (entry *Entry) Set(field Field, val interface{}) {
	field.Set(entry, val, true)
}

func (entry *Entry) Delete(field Field) (interface{}, bool) {
	return field.Delete(entry)
}

func (entry *Entry) Read(field Field, dest interface{}) error {
	val, ok := entry.Get(field)
	if !ok {
		return fmt.Errorf("Field does not exist")
	}

	switch dest := dest.(type) {
	case *string:
		if str, ok := val.(string); ok {
			*dest = str
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to string", field, val)
		}
	case *map[string]interface{}:
		if m, ok := val.(map[string]interface{}); ok {
			*dest = m
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to map[string]interface{}", field, val)
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
