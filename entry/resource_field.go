package entry

import "fmt"

// ResourceField is the path to an entry's resource key
type ResourceField struct {
	key string
}

// Get will return the resource value and a boolean indicating if it exists
func (l ResourceField) Get(entry *Entry) (interface{}, bool) {
	if entry.Resource == nil {
		return "", false
	}
	val, ok := entry.Resource[l.key]
	return val, ok
}

// Set will set the resource value on an entry
func (l ResourceField) Set(entry *Entry, val interface{}) error {
	if entry.Resource == nil {
		entry.Resource = make(map[string]string, 1)
	}

	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("cannot set a resource to a non-string value")
	}
	entry.Resource[l.key] = str
	return nil
}

// Delete will delete a resource key from an entry
func (l ResourceField) Delete(entry *Entry) (interface{}, bool) {
	if entry.Resource == nil {
		return "", false
	}

	val, ok := entry.Resource[l.key]
	delete(entry.Resource, l.key)
	return val, ok
}

func (l ResourceField) String() string {
	return "$resource." + l.key
}

// NewResourceField will creat a new resource field from a key
func NewResourceField(key string) Field {
	return Field{ResourceField{key}}
}
