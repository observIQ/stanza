## `azure_log_analytics_input` operator

The `azure_log_analytics_input` operator reads Azure Log Analytics logs from Azure Event Hub using.

The `azure_log_analytics_input` operator will use the `timegenerated` field as the parsed entry's timestamp. The label `azure_log_analytics_type` is derived from the log's `type` field.  All other fields are added to the entry's record.

## Prerequisites

You must define a Log Analytics Export Rule using Azure CLI. Microsoft has documentation [here](https://docs.microsoft.com/en-us/azure/azure-monitor/logs/logs-data-export?tabs=portal)

### Configuration Fields

| Field               | Default                | Description                                                                                   |
| ---                 | ---                    | ---                                                                                           |
| `id`                | `azure_log_analytics_input` | A unique identifier for the operator                                                          |
| `output`            | Next in pipeline       | The connected operator(s) that will receive all outbound entries                              |
| `namespace`         | required               | The Event Hub Namespace                                                                       |
| `name`              | required               | The Event Hub Name                                                                            |
| `group`             | required               | The Event Hub Consumer Group                                                                  |
| `connection_string` | required               | The Event Hub [connection string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string) |
| `prefetch_count`    | `1000`                 | Desired number of events to read at one time                                                  |
| `start_at`          | `end`                  | At startup, where to start reading events. Options are `beginning` or `end`                   |

### Example Configurations

#### Simple Azure Event Hub input

Configuration:
```yaml
pipeline:
- type: azure_log_analytics_input
  namespace: stanza
  name: devel
  group: Default
  connection_string: 'Endpoint=sb://stanza.servicebus.windows.net/;SharedAccessKeyName=dev;SharedAccessKey=supersecretkey;EntityPath=devel'
  start_at: end
```

### Example Output

A list of potential fields for each Azure Log Analytics table can be found [here](https://docs.microsoft.com/en-us/azure/azure-monitor/reference/tables/tables-category).

```json
{
  "timestamp": "2021-04-26T18:19:31.358Z",
  "severity": 0,
  "labels": {
    "azure_log_analytics_type": "ContainerLog",
  },
  "record": {
    "_internal_workspaceresourceid": "/subscriptions/09373b6b-bc8b-4093-925d-eb87334c7d56/resourcegroups/bindplane-integration/providers/microsoft.operationalinsights/workspaces/bp-integration1",
    "_resourceid": "/subscriptions/09373b6b-bc8b-4093-925d-eb87334c7d56/resourceGroups/devops/providers/Microsoft.ContainerService/managedClusters/log-analytics",
    "computer": "aks-agentpool-39365618-vmss000001",
    "containerid": "93f4537223ae81d1c39e12e684de25c65207549d1003d153356055a6137f82b0",
    "logentry": "[SpanData(name='Recv.grpc.health.v1.Health.Check', context=SpanContext(trace_id=9d186b35325a4a9093242435948ada22, span_id=None, trace_options=TraceOptions(enabled=True), tracestate=None), span_id='bc4877b54bc8407b', parent_span_id=None, attributes={'component': 'grpc'}, start_time='2021-04-26T18:19:31.358155Z', end_time='2021-04-26T18:19:31.358229Z', child_span_count=0, stack_trace=None, time_events=[<opencensus.trace.time_event.TimeEvent object at 0x7f5d9fc53190>, <opencensus.trace.time_event.TimeEvent object at 0x7f5d9fc537d0>], links=[], status=None, same_process_as_parent_span=None, span_kind=1)]",
    "logentrysource": "stdout",
    "mg": "00000000-0000-0000-0000-000000000002",
    "sourcesystem": "Containers",
    "system_properties": {
      "x-opt-enqueued-time": "2021-04-26T18:19:50.361Z",
      "x-opt-offset": 14480072,
      "x-opt-sequence-number": 1548
    },
    "tenantid": "ae0db88b-40bb-40b7-b056-57980214436c",
    "timegenerated": "2021-04-26T18:19:31.3580000Z",
    "timeofcommand": "2021-04-26T18:19:44.0000000Z"
  }
}
```