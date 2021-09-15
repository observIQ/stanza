package loganalytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator/builtin/input/azure"
	"go.uber.org/zap"
)

// handleBatchedEvents handles an event recieved by an Event Hub consumer.
func (l *LogAnalyticsInput) handleBatchedEvents(ctx context.Context, event *azhub.Event) error {
	l.WG.Add(1)
	defer l.WG.Done()

	type record struct {
		Records []map[string]interface{} `json:"records"`
	}

	// Create a "base" event by capturing the batch log records from the event's Data field.
	// If Unmarshalling fails, fallback on handling the event as a single log entry.
	records := record{}
	if err := json.Unmarshal(event.Data, &records); err != nil {
		id := event.ID
		if id == "" {
			id = "unknown"
		}
		l.Warnw(fmt.Sprintf("Failed to parse event '%s' as JSON. Expcted key 'records' in event.Data.", string(event.Data)), zap.Error(err))
		l.handleEvent(ctx, *event, nil)
		return nil
	}
	event.Data = nil

	// Create an entry for each log in the batch, using the origonal event's fields
	// as a starting point for each entry
	wg := sync.WaitGroup{}
	max := 10
	gaurd := make(chan struct{}, max)
	for i := 0; i < len(records.Records); i++ {
		r := records.Records[i]
		wg.Add(1)
		gaurd <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-gaurd
			}()
			l.handleEvent(ctx, *event, r)
		}()
	}
	wg.Wait()
	return nil
}

func (l *LogAnalyticsInput) handleEvent(ctx context.Context, event azhub.Event, records map[string]interface{}) {
	e, err := l.NewEntry(nil)
	if err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}

	if err = l.parse(event, records, e); err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}
	l.Write(ctx, e)
}

// parse returns an entry from an event and set of records
func (l *LogAnalyticsInput) parse(event azhub.Event, records map[string]interface{}, e *entry.Entry) error {
	// make sure all keys are lower case
	for k, v := range records {
		delete(records, k)
		records[strings.ToLower(k)] = v
	}

	// Add base fields shared among all log records from the event
	err := azure.ParseEvent(event, e)
	if err != nil {
		return err
	}

	// set label azure_log_analytics_table
	records, err = l.setType(e, records)
	if err != nil {
		return err
	}

	if err := l.setTimestamp(e, records); err != nil {
		return err
	}

	// Add remaining records to record.<azure_log_analytics_table> map
	return l.setField(e, e.Labels["azure_log_analytics_table"], records)
}

// setType sets the label 'azure_log_analytics_table'
func (l *LogAnalyticsInput) setType(e *entry.Entry, records map[string]interface{}) (map[string]interface{}, error) {
	const typeField = "type"

	for key, value := range records {
		switch key {
		case typeField:
			if v, ok := value.(string); ok {
				v = strings.ToLower(v)

				// Set the log table label
				if err := l.setLabel(e, "azure_log_analytics_table", v); err != nil {
					return nil, err
				}

				delete(records, key)
				return records, nil
			}
			return nil, fmt.Errorf("expected '%s' field to be a string", typeField)
		}
	}
	return nil, fmt.Errorf("expected to find field with name '%s'", typeField)
}

// setTimestamp set the entry's timestamp using the timegenerated log analytics field
func (l *LogAnalyticsInput) setTimestamp(e *entry.Entry, records map[string]interface{}) error {
	for key, value := range records {
		switch key {
		case "timegenerated":
			if v, ok := value.(string); ok {
				t, err := time.Parse("2006-01-02T15:04:05.0000000Z07", v)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("failed to promote timestamp from %s field", key))
				}
				e.Timestamp = t
				return nil
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
	r := entry.BodyField{
		Keys: []string{key},
	}
	return r.Set(e, value)
}
