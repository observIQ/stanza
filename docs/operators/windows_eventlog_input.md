## `windows_eventlog_input` operator

The `windows_eventlog_input` operator reads logs from the windows event log API.

### Configuration Fields

| Field             | Default                  | Description                                                                                  |
| ---               | ---                      | ---                                                                                          |
| `id`              | `windows_eventlog_input` | A unique identifier for the operator                                                         |
| `output`          | Next in pipeline         | The connected operator(s) that will receive all outbound entries                             |
| `channel`         | required                 | The windows event log channel to monitor                                                     |
| `max_reads`       | 100                      | The maximum number of records read and processed at one time                                 |
| `start_at`        | `end`                    | On first startup, where to start reading logs from the API. Options are `beginning` or `end` |
| `poll_interval`   | 1s                       | The interval at which the channel is checked for new log entries                             |
| `log_type`        | `windows_eventlog_input` | The log_type label appended to all discovered entries                                        |
| `append_log_type` | `true`                   | If true, appends the log_type label to all entries                                           |

### Example Configurations

#### Simple

Configuration:
```yaml
- type: windows_eventlog_input
  channel: application
  start_at: beginning
```

Output entry sample:
```json
{
  "timestamp": "2020-04-30T12:10:17.656726-04:00",
  "severity": 30,
  "record": {
		"provider_name": "example provider",
		"provider_id": "provider guid",
		"event_source": "example source",
		"system_time": "2020-04-30T12:10:17.656726789Z",
		"computer": "example computer",
		"channel": "application",
		"record_id": 1,
		"level": "Information",
		"message": "example message",
		"task": "example task",
		"opcode": "example opcode",
		"keywords": ["example keyword"],
	}
}
```
