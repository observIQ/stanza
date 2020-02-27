package entry

import "time"

type Entry struct {
	Timestamp time.Time
	// TODO consider using a more allocation-efficient representation
	Record map[string]interface{}
}
