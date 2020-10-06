package windows

import (
	"encoding/xml"
	"fmt"
	"strings"
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
	if details != nil {
		record["details"] = details
	}
	return record
}

// parseMessage will attempt to parse a message into a message and details
func (e *EventXML) parseMessage() (string, map[string]interface{}) {

	// Other channels will be parsed in a later revision
	if e.Channel != "Security" {
		return e.Message, nil
	}

	if e.Message == "" {
		return "", nil
	}

	sections := strings.Split(e.Message, "\n\n")

	if len(sections) == 1 {
		return e.Message, nil
	}

	message := sections[0]

	details := map[string]interface{}{}
	moreInfo := []string{}
	unparsed := []string{}
	for i := 1; i < len(sections); i++ {

		lines := strings.Split(sections[i], "\n")
		if len(lines) == 1 {
			/*
				String
				or
				Key: Value
			*/
			keyVal := strings.Split(lines[0], ":")
			if len(keyVal) == 1 {
				// String
				moreInfo = append(moreInfo, strings.TrimSpace(lines[0]))
			} else if len(keyVal) == 2 {
				// Key: Value
				details[strings.TrimSpace(keyVal[0])] = strings.TrimSpace(keyVal[1])
			} else {
				// Unexpected format
				unparsed = append(unparsed, strings.TrimSpace(lines[0]))
			}
		} else if strings.Contains(lines[0], ":") {
			keyVal := strings.Split(strings.TrimSpace(lines[0]), ":")

			if len(keyVal) == 1 || strings.TrimSpace(keyVal[1]) == "" {
				/*
					Key:
						Key1:	Value1
						Key2:	Value2
				*/
				m := map[string]string{}
				for j := 1; j < len(lines); j++ {
					kv := strings.Split(lines[j], ":")
					if len(kv) == 1 || strings.TrimSpace(kv[1]) == "" {
						m[strings.TrimSpace(kv[0])] = "-"
					} else {
						m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
					}
				}
				details[strings.TrimSpace(keyVal[0])] = m
			} else if len(keyVal) == 2 {
				/*
					Key:		Item1
								Item2
				*/
				a := []string{strings.TrimSpace(keyVal[1])}
				for j := 1; j < len(lines); j++ {
					v := strings.TrimSpace(lines[j])
					if v != "" {
						a = append(a, v)
					}
				}
				details[strings.TrimSpace(keyVal[0])] = a
			} else {
				// Unexpected format
				for j := range lines {
					unparsed = append(unparsed, strings.TrimSpace(lines[j]))
				}
			}
		} else {
			for j := range lines {
				moreInfo = append(moreInfo, strings.TrimSpace(lines[j]))
			}
		}
	}
	if len(moreInfo) > 0 {
		details["Additional Context"] = moreInfo
	}

	if len(unparsed) > 0 {
		details["Unparsed"] = unparsed
	}

	return message, details
}

// unmarshalEventXML will unmarshal EventXML from xml bytes.
func unmarshalEventXML(bytes []byte) (EventXML, error) {
	var eventXML EventXML
	if err := xml.Unmarshal(bytes, &eventXML); err != nil {
		return EventXML{}, fmt.Errorf("failed to unmarshal xml bytes into event: %s", err)
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
