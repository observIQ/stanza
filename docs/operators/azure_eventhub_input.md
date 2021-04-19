## `azure_eventhub_input` operator

The `azure_eventhub_input` operator reads logs from Azure Event Hub using [Azure's SDK](https://github.com/Azure/azure-event-hubs-go)

The `azure_eventhub_input` operator will use the `EnqueuedTime` field of the event as the parsed entry's timestamp. If `EnqueuedTime` is not set, `azure_eventhub_input` will use `IoTHubEnqueuedTime` if it is set. All other fields are added to the entry's record.

### Configuration Fields

| Field               | Default                | Description                                                                                   |
| ---                 | ---                    | ---                                                                                           |
| `id`                | `azure_eventhub_input` | A unique identifier for the operator                                                          |
| `output`            | Next in pipeline       | The connected operator(s) that will receive all outbound entries                              |
| `namespace`         |                        | The Event Hub Namespace                                                                       |
| `name`              |                        | The Event Hub Name                                                                            |
| `group`             |                        | The Event Hub Consumer Group                                                                  |
| `connection_string` |                        | The Event Hub [connection string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string) |
| `prefetch_count`    | `1000`                 | Desired number of events to read at one time                                                  |
| `start_at`          | `end`                  | At startup, where to start reading events. Options are `beginning` or `end`                   |

### Example Configurations

#### Simple Azure Event Hub input

Configuration:
```yaml
pipeline:
- type: azure_eventhub_input
  namespace: stanza
  name: devel
  group: Default
  connection_string: 'Endpoint=sb://stanza.servicebus.windows.net/;SharedAccessKeyName=dev;SharedAccessKey=supersecretkey;EntityPath=devel'
  start_at: end
```

### Example Output

A list of potential keys and their purpose can be found [here](https://github.com/Azure/azure-event-hubs-go/blob/master/event.go). Event Hub `system_properties` documentation can be found [here](https://docs.microsoft.com/en-us/azure/data-explorer/ingest-data-event-hub-overview#event-system-properties-mapping)

```json
{
  "timestamp": "2021-04-19T18:44:34.619Z",
  "severity": 0,
  "record": {
    "data": "hello, world!",
    "id": "28447813-ebac-4908-9bd4-c1a79f9f98ad",
    "system_properties": {
      "x-opt-enqueued-time": "2021-04-19T18:44:34.619Z",
      "x-opt-offset": 6120,
      "x-opt-sequence-number": 51
    }
  }
}
```