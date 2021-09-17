## `azure_log_analytics_input` operator

The `azure_log_analytics_input` operator reads Azure Log Analytics logs from Azure Event Hub using.

The `azure_log_analytics_input` operator will use the `timegenerated` field as the parsed entry's timestamp. The attribute `azure_log_analytics_table` is derived from the log's `type` field.

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
  "timestamp": "2021-05-07T14:01:26.105Z",
  "severity": 0,
  "record": {
    "containerlog": {
      "_internal_workspaceresourceid": "/subscriptions/000-000/resourcegroups/integration/providers/microsoft.operationalinsights/workspaces/stanza",
      "_resourceid": "/subscriptions/0000-000/resourceGroups/devops/providers/Microsoft.ContainerService/managedClusters/log-analytics",
      "computer": "aks-agentpool-39365618-vmss000001",
      "containerid": "f5376c6972ac19630113736e7d3bf359fe67065fde3831b0502cfee33470e68f",
      "logentry": "request to api failed"
      "logentrysource": "stdout",
      "mg": "00000000-0000-0000-0000-000000000002",
      "sourcesystem": "Containers",
      "tenantid": "ae0db88b-40bb-40b7-b056-57980214436c",
      "timegenerated": "2021-05-07T14:01:26.1050000Z",
      "timeofcommand": "2021-05-07T14:01:29.0000000Z"
    },
    "system_properties": {
      "x-opt-enqueued-time": "2021-05-07T14:01:37.789Z",
      "x-opt-offset": 150347296000,
      "x-opt-sequence-number": 125576
    }
  }
}
```
