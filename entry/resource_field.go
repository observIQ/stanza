package entry

import (
	"fmt"
	"strings"
)

// ResourceField is the path to an entry's resource key
type ResourceField struct {
	key string
}

// Get will return the resource value and a boolean indicating if it exists
func (r ResourceField) Get(entry *Entry) (interface{}, bool) {
	if entry.Resource == nil {
		return "", false
	}
	val, ok := entry.Resource[r.key]
	return val, ok
}

// Set will set the resource value on an entry
func (r ResourceField) Set(entry *Entry, val interface{}) error {
	if entry.Resource == nil {
		entry.Resource = make(map[string]string, 1)
	}

	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("cannot set a resource to a non-string value")
	}
	entry.Resource[r.key] = str
	return nil
}

// Delete will delete a resource key from an entry
func (r ResourceField) Delete(entry *Entry) (interface{}, bool) {
	if entry.Resource == nil {
		return "", false
	}

	val, ok := entry.Resource[r.key]
	delete(entry.Resource, r.key)
	return val, ok
}

func (r ResourceField) String() string {
	if strings.Contains(r.key, ".") {
		return fmt.Sprintf(`$resource['%s']`, r.key)
	}
	return "$resource." + r.key
}

// NewResourceField will creat a new resource field from a key
func NewResourceField(key string) Field {
	return Field{ResourceField{key}}
}
