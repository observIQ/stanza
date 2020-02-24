package entry

import "time"

type Entry struct {
	Timestamp time.Time
	Record    map[string]interface{}
}
