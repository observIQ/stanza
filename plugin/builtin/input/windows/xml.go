// +build windows

package windows

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/observiq/carbon/entry"
)

// EventXML is an event stored as xml in windows event log.
type EventXML struct {
	Provider    Provider    `xml:"System>Provider"`
	Computer    string      `xml:"System>Computer"`
	Channel     string      `xml:"System>Channel"`
	RecordID    uint64      `xml:"System>EventRecordID"`
	TimeCreated TimeCreated `xml:"System>TimeCreated"`
	Message     string      `xml:"RenderingInfo>Message"`
	Level       string      `xml:"RenderingInfo>Level"`
	Task        string      `xml:"RenderingInfo>Task"`
	Opcode      string      `xml:"RenderingInfo>Opcode"`
	Keywords    []string    `xml:"RenderingInfo>Keywords>Keyword"`
}

// parseTimestamp will parse the timestamp of the event.
func (e *EventXML) parseTimestamp() time.Time {
	if timestamp, err := time.Parse(time.RFC3339Nano, e.TimeCreated.SystemTime); err == nil {
		return timestamp
	}
	return time.Now()
}

// parseSeverity will parse the severity of the event.
func (e *EventXML) parseSeverity() entry.Severity {
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
func (e *EventXML) parseRecord() map[string]interface{} {
	return map[string]interface{}{
		"provider_name": e.Provider.Name,
		"provider_id":   e.Provider.GUID,
		"event_source":  e.Provider.EventSourceName,
		"computer":      e.Computer,
		"channel":       e.Channel,
		"record_id":     e.RecordID,
		"message":       e.Message,
		"task":          e.Task,
		"opcode":        e.Opcode,
		"keywords":      e.Keywords,
	}
}

// ToEntry will convert the event xml into a entry.
func (e *EventXML) ToEntry() *entry.Entry {
	return &entry.Entry{
		Timestamp: e.parseTimestamp(),
		Severity:  e.parseSeverity(),
		Record:    e.parseRecord(),
	}
}

// unmarshalEventXML will unmarshal EventXML from xml bytes.
func unmarshalEventXML(bytes []byte) (EventXML, error) {
	var eventXML EventXML
	if err := xml.Unmarshal(bytes, &eventXML); err != nil {
		return EventXML{}, fmt.Errorf("failed to unmarshal xml bytes into event: %s", err)
	}
	return eventXML, nil
}

// TimeCreated is the creation time of the event.
type TimeCreated struct {
	SystemTime string `xml:"SystemTime,attr"`
}

// Provider is the provider of the event.
type Provider struct {
	Name            string `xml:"Name,attr"`
	GUID            string `xml:"Guid,attr"`
	EventSourceName string `xml:"EventSourceName,attr"`
}
