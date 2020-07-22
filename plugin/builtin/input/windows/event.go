// +build windows

package windows

import (
	"time"

	"github.com/observiq/carbon/entry"
)

// Event is an event rendered from the windows event log api.
type Event struct {
	Computer   string   `xml:"System>Computer"`
	Channel    string   `xml:"System>Channel"`
	RecordID   uint64   `xml:"System>EventRecordID"`
	SystemTime string   `xml:"System>TimeCreated>SystemTime,attr"`
	Message    string   `xml:"RenderingInfo>Message"`
	Level      string   `xml:"RenderingInfo>Level"`
	Task       string   `xml:"RenderingInfo>Task"`
	Opcode     string   `xml:"RenderingInfo>Opcode"`
	Keywords   []string `xml:"RenderingInfo>Keywords>Keyword"`
}

// parseTimestamp will parse the timestamp of the event.
func (e *Event) parseTimestamp() time.Time {
	if timestamp, err := time.Parse(time.RFC3339Nano, e.SystemTime); err != nil {
		return timestamp
	}
	return time.Now()
}

// parseSeverity will parse the severity of the event.
func (e *Event) parseSeverity() entry.Severity {
	switch e.Level {
	case "Critical":
		return entry.Critical
	case "Error":
		return entry.Error
	case "Warning":
		return entry.Warning
	case "Informational":
		return entry.Info
	default:
		return entry.Default
	}
}

// parseRecord will parse a record from the event.
func (e *Event) parseRecord() map[string]interface{} {
	return map[string]interface{}{
		"computer":  e.Computer,
		"channel":   e.Channel,
		"record_id": e.RecordID,
		"message":   e.Message,
		"task":      e.Task,
		"opcode":    e.Opcode,
		"keywords":  e.Keywords,
	}
}

// ToEntry will convert the event into a carbon entry.
func (e *Event) ToEntry() *entry.Entry {
	return &entry.Entry{
		Timestamp: e.parseTimestamp(),
		Severity:  e.parseSeverity(),
		Record:    e.parseRecord(),
	}
}
