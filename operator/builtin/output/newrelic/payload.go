package newrelic

import (
	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/version"
)

// LogPayloadFromEntries creates a new []*LogPayload from an array of entries
func LogPayloadFromEntries(entries []*entry.Entry, messageField entry.Field) LogPayload {
	logs := make([]*LogMessage, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, LogMessageFromEntry(entry, messageField))
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
func LogMessageFromEntry(entry *entry.Entry, messageField entry.Field) *LogMessage {
	logMessage := &LogMessage{
		Timestamp:  entry.Timestamp.UnixNano() / 1000 / 1000, // Convert to millis
		Attributes: make(map[string]interface{}),
	}

	var message string
	err := entry.Read(messageField, &message)
	if err == nil {
		logMessage.Message = message
	}

	logMessage.Attributes["record"] = entry.Record
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
