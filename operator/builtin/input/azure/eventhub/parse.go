package eventhub

import (
	"reflect"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/observiq/stanza/entry"
)

// parse parses an Azure Event Hub event as an Entry. Nil values are
// dropped from the record.
func parse(event *azhub.Event) (*entry.Entry, error) {
	m := make(map[string]interface{})
	e := entry.New()

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
	if err := mapstructure.Decode(event.SystemProperties, &sysProp); err != nil {
		return e, err
	}
	for key, _ := range sysProp {
		if sysProp[key] == nil || reflect.ValueOf(sysProp[key]).IsNil() {
			delete(sysProp, key)
		}
	}
	m["system_properties"] = sysProp

	e.Record = m
	return e, nil
}
