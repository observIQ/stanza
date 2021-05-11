## `aws_cloudwatch_input` operator

The `aws_cloudwatch_input` operator reads logs from AWS Cloudwatch Logs using [AWS's SDK](https://github.com/aws/aws-sdk-go)

The `aws_cloudwatch_input` operator will use the `Timestamp` field of the event as the parsed entry's timestamp. All other fields are added to the entry's record.

The `aws_cloudwatch_input` operator will use the following order to get credentials. Environment Variables, Shared Credentials file, Shared Configuration file (if SharedConfig is enabled), and EC2 Instance Metadata (credentials only). You can provide `profile` to specify which credential set to use from a Shared Credentials file.

### Configuration Fields

| Field                     | Default                | Description                                                                                   |
| ---                       | ---                    | ---                                                                                           |
| `id`                      | `aws_cloudwatch_input` | A unique identifier for the operator                                                          |
| `output`                  | Next in pipeline       | The connected operator(s) that will receive all outbound entries                              |
| `LogGroupName`            | required               | The Cloudwatch Logs Log Group Name                                                            |
| `Region`                  | required               | The AWS Region to be used.                                                                    |
| `log_stream_name_prefix`  |                        | The log stream name prefix to use. This will find any log stream name in the group with the starting prefix |
| `log_stream_names`        |                        | The Event Hub [connection string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string) |
| `profile`                 |                        | Desired number of events to read at one time                                                  |
| `event_limit`             | `10000`                | Desired number of events to read at one time                                                  |
| `poll_interval`           | `1m`                   | Desired number of events to read at one time                                                  |
| `start_at`                | `end`                  | At startup, where to start reading events. Options are `beginning` or `end`                   |

### Log Stream Prefix

The log_stream_prefix allows the use of "directives" such as `%Y` (4-digit year) and `%d` (2-digit zero-padded day). These directives are based on `strptime` directives. There are a limited set of the `strptime` directives. These directives are listed below. When directive is detected within the prefix it will replace the first occurance of directive.

#### Supported directives

| Directive | Description                        |
| :---:     | :---                               |
| %Y        | Year, zero-padded                  |
| %y        | Year, last two digits, zero-padded |
| %m        | Month, zero-padded                 |
| %q        | Month as a unpadded number         |
| %b        | Abbreviated month name             |
| %h        | Abbreviated month name             |
| %B        | Full month name                    |
| %d        | Day of the month, zero-padded      |
| %g        | Day of the month, unpadded         |
| %a        | Abbreviated weekday name           |
| %A        | Full weekday name                  |


### Example Configurations

#### Simple Azure Event Hub input

Configuration:
```yaml
pipeline:
- type: aws_cloudwatch_input
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
  "resource": {
    "event_id": "fea3c182-00a6-4951-8f6f-9331031f978f"
  },
  "record": {
    "message": "hello, world!",
    "system_properties": {
      "x-opt-enqueued-time": "2021-04-19T18:44:34.619Z",
      "x-opt-offset": 6120,
      "x-opt-sequence-number": 51
    }
  }
}
```
