package newrelic

import (
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/internal/version"
)

// LogPayloadFromEntries creates a new []*LogPayload from an array of entries
func LogPayloadFromEntries(entries []*entry.Entry) LogPayload {
	logs := make([]*LogMessage, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, LogMessageFromEntry(entry))
	}

	lp := LogPayload{{
		Common: LogPayloadCommon{
			Attributes: map[string]interface{}{
				"plugin": map[string]interface{}{
					"type":    "stanza",
					"version": version.GetVersion(),
				},
			},
		},
		Logs: logs,
	}}

	return lp
}

// LogPayload represents a single payload delivered to the New Relic Log API
type LogPayload []struct {
	Common LogPayloadCommon `json:"common"`
	Logs   []*LogMessage    `json:"logs"`
}

// LogPayloadCommon represents the common attributes in a payload segment
type LogPayloadCommon struct {
	// Milliseconds or seconds since epoch
	Timestamp int `json:"timestamp,omitempty"`

	Attributes map[string]interface{} `json:"attributes"`
}

// LogMessageFromEntry creates a new LogMessage from a given entry.Entry
func LogMessageFromEntry(entry *entry.Entry) *LogMessage {
	logMessage := &LogMessage{
		Timestamp:  entry.Timestamp.UnixNano() / 1000 / 1000, // Convert to millis
		Attributes: make(map[string]interface{}),
	}

	// Promote message to the top-level
	switch r := entry.Record.(type) {
	case string:
		logMessage.Message = r
	case []byte:
		logMessage.Message = string(r)
	case map[string]interface{}:
		// Instead of modifying the original with delete(), copy
		// each key/value, promoting message if we come across it
		for k, v := range r {
			if k != "message" {
				logMessage.Attributes[k] = v
			} else if msgString, ok := v.(string); ok {
				logMessage.Message = msgString
			} else if msgBytes, ok := v.([]byte); ok {
				logMessage.Message = string(msgBytes)
			}
		}
	case map[string]string:
		for k, v := range r {
			if k != "message" {
				logMessage.Attributes[k] = v
			} else {
				logMessage.Message = v
			}
		}
	}

	logMessage.Attributes["resource"] = entry.Resource
	logMessage.Attributes["labels"] = entry.Labels
	logMessage.Attributes["severity"] = entry.Severity.String()

	return logMessage
}

// LogMessage represents a single log entry that will be marshalled
// in the format expected by the New Relic Log API
type LogMessage struct {
	// Milliseconds or seconds since epoch
	Timestamp  int64                  `json:"timestamp,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Message    string                 `json:"message,omitempty"`
}
