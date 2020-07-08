package entry

import "fmt"

type LabelField struct {
	key string
}

func (l LabelField) Get(entry *Entry) (interface{}, bool) {
	if entry.Labels == nil {
		return "", false
	}
	val, ok := entry.Labels[l.key]
	return val, ok
}

func (l LabelField) Set(entry *Entry, val interface{}) error {
	if entry.Labels == nil {
		entry.Labels = make(map[string]string, 1)
	}

	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("cannot set a label to a non-string value")
	}
	entry.Labels[l.key] = str
	return nil
}

func (l LabelField) Delete(entry *Entry) (interface{}, bool) {
	if entry.Labels == nil {
		return "", false
	}

	val, ok := entry.Labels[l.key]
	delete(entry.Labels, l.key)
	return val, ok
}

func (l LabelField) String() string {
	return "$labels." + l.key
}

func NewLabelField(key string) Field {
	return Field{LabelField{key}}
}
