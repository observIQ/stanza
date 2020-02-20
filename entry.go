package bplogagent

import "time"

type Entry struct {
	Timestamp time.Time
	Record    map[string]interface{}
}
