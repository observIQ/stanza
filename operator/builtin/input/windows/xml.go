package windows

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/observiq/stanza/entry"
)

// EventXML is the rendered xml of an event.
type EventXML struct {
	EventID     EventID     `xml:"System>EventID"`
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
	case "Information":
		return entry.Info
	default:
		return entry.Default
	}
}

// parseRecord will parse a record from the event.
func (e *EventXML) parseRecord() map[string]interface{} {
	message, details := e.parseMessage()
	record := map[string]interface{}{
		"event_id": map[string]interface{}{
			"qualifiers": e.EventID.Qualifiers,
			"id":         e.EventID.ID,
		},
		"provider": map[string]interface{}{
			"name":         e.Provider.Name,
			"guid":         e.Provider.GUID,
			"event_source": e.Provider.EventSourceName,
		},
		"system_time": e.TimeCreated.SystemTime,
		"computer":    e.Computer,
		"channel":     e.Channel,
		"record_id":   e.RecordID,
		"level":       e.Level,
		"message":     message,
		"task":        e.Task,
		"opcode":      e.Opcode,
		"keywords":    e.Keywords,
	}
	if len(details) > 0 {
		record["details"] = details
	}
	return record
}

// parseMessage will attempt to parse a message into a message and details
func (e *EventXML) parseMessage() (string, map[string]interface{}) {
	switch e.Channel {
	case "Security":
		return parseSecurity(e.Message)
	default:
		return e.Message, nil
	}
}

// unmarshalEventXML will unmarshal EventXML from xml bytes.
func unmarshalEventXML(bytes []byte) (EventXML, error) {
	var eventXML EventXML
	if err := xml.Unmarshal(bytes, &eventXML); err != nil {
		return EventXML{}, fmt.Errorf("failed to unmarshal xml bytes into event: %w (%s)", err, string(bytes))
	}
	return eventXML, nil
}

// EventID is the identifier of the event.
type EventID struct {
	Qualifiers uint16 `xml:"Qualifiers,attr"`
	ID         uint32 `xml:",chardata"`
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
