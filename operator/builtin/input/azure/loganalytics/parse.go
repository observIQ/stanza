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
