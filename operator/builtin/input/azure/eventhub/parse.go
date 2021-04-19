package eventhub

import (
	"reflect"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/observiq/stanza/entry"
)

// parseEvent parses an Azure Event Hub event as an Entry.
func parseEvent(event *azhub.Event) (*entry.Entry, error) {
	e := entry.New()

	x, err := parse(event)
	if err != nil {
		return e, err
	}

	promoteTime(event, e)

	e.Record = x
	return e, nil
}

// parse parses an Azure Event Hub event into a map. Nil values
// are dropped. Keys 'data' and 'id' are gauranteed to be present.
func parse(event *azhub.Event) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	m["data"] = ""
	if len(event.Data) > 0 {
		m["data"] = string(event.Data)
	}

	if event.PartitionKey != nil {
		m["partition_key"] = event.PartitionKey
	}

	if event.Properties != nil {
		m["properties"] = event.Properties
	}

	m["id"] = event.ID

	sysProp := make(map[string]interface{})
	if event.SystemProperties != nil {
		if err := mapstructure.Decode(event.SystemProperties, &sysProp); err != nil {
			return m, err
		}
		for key, _ := range sysProp {
			if sysProp[key] == nil || reflect.ValueOf(sysProp[key]).IsNil() {
				delete(sysProp, key)
			}
		}
		m["system_properties"] = sysProp
	}

	return m, nil
}

// promoteTime promotes an Azure Event Hub event's timestamp
// EnqueuedTime takes precedence over IoTHubEnqueuedTime
func promoteTime(event *azhub.Event, e *entry.Entry) {
	timestamps := []*time.Time{
		event.SystemProperties.EnqueuedTime,
		event.SystemProperties.IoTHubEnqueuedTime,
	}

	for _, t := range timestamps {
		if t != nil && !t.IsZero() {
			e.Timestamp = *t
			return
		}
	}
}
