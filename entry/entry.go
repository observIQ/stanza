package entry

import (
	"fmt"
	"time"
)

// Entry is a flexible representation of log data associated with a timestamp.
type Entry struct {
	Timestamp time.Time   `json:"timestamp"`
	Record    interface{} `json:"record"`
}

// New will create a new log entry with current timestamp and an empty record.
func New() *Entry {
	return &Entry{
		Timestamp: time.Now(),
		Record:    map[string]interface{}{},
	}
}

func (entry *Entry) Get(path Field) (interface{}, bool) {
	return path.Get(entry)
}

func (entry *Entry) Set(path Field, val interface{}) {
	path.Set(entry, val)
}

func (entry *Entry) Delete(path Field) (interface{}, bool) {
	return path.Delete(entry)
}

func (entry *Entry) Read(path Field, dest interface{}) error {
	val, ok := entry.Get(path)
	if !ok {
		return fmt.Errorf("Field does not exist")
	}

	switch dest := dest.(type) {
	case *string:
		if str, ok := val.(string); ok {
			*dest = str
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to string", path, val)
		}
	case *map[string]interface{}:
		if m, ok := val.(map[string]interface{}); ok {
			*dest = m
		} else {
			return fmt.Errorf("can not cast field '%s' of type '%T' to map[string]interface{}", path, val)
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
