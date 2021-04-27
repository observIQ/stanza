package loganalytics

import (
	"fmt"
	"strings"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator/builtin/input/azure"
)

// parse returns an entry from an event and set of records
func (l *LogAnalyticsInput) parse(event azhub.Event, records map[string]interface{}, e *entry.Entry) error {
	// Add base fields shared among all log records from the event
	if err := azure.ParseEvent(event, e); err != nil {
		return err
	}

	// Add key value pairs from Log Analytics log to entry's resources and record
	for key, value := range records {
		// Promote resources
		switch k := strings.ToLower(key); k {
		case "type":
			if v, ok := value.(string); ok {
				key := "azure_log_analytics_type"
				if err := l.setLabel(e, key, v); err != nil {
					return err
				}
			}
		case "timegenerated":
			if v, ok := value.(string); ok {
				t, err := time.Parse("2006-01-02T15:04:05.0000000Z07", v)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("failed to promote timestamp from %s field", k))
				}

				// set as timestamp and preserve the field
				e.Timestamp = t
				if err := l.setField(e, k, value); err != nil {
					return err
				}
			}
		// All other keys are fields
		default:
			if err := l.setField(e, k, value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *LogAnalyticsInput) setResource(e *entry.Entry, key, value string) {
	e.AddResourceKey(key, value)
}

func (l *LogAnalyticsInput) setLabel(e *entry.Entry, key string, value interface{}) error {
	r := entry.NewLabelField(key)
	return r.Set(e, value)
}

func (l *LogAnalyticsInput) setField(e *entry.Entry, key string, value interface{}) error {
	r := entry.RecordField{
		Keys: []string{key},
	}
	return r.Set(e, value)
}
