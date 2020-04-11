package entry

import "time"

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

// func (entry *Entry) Merge(selector FieldSelector, val map[string]interface{}) {
// 	selector.Merge(entry.Record, val)
// }
