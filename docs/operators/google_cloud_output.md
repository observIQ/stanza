## `google_cloud_output` operator

The `google_cloud_output` operator will send entries to Google Cloud Logging.

### Configuration Fields

| Field              | Default               | Description                                                                                                |
| ---                | ---                   | ---                                                                                                        |
| `id`               | `google_cloud_output` | A unique identifier for the operator                                                                       |
| `credentials`      |                       | The JSON-formatted credentials for the logs writer service account                                         |
| `credentials_file` |                       | A path to a file containing the JSON-formatted credentials                                                 |
| `project_id`       |                       | The Google Cloud project ID the logs should be sent to. Defaults to project_id found in credentials        |
| `log_name_field`   |                       | A [field](/docs/types/field.md) for the log name on the entry. Log name defaults to `default` if unset     |
| `location_field`   |                       | A [field](/docs/types/field.md) for the log location resource when an entry [fulfills a monitored resource type's requirements](https://cloud.google.com/logging/docs/api/v2/resource-list#resource-types) |
| `severity_field`   |                       | A [field](/docs/types/field.md) for the severity on the log entry                                          |
| `trace_field`      |                       | A [field](/docs/types/field.md) for the trace on the log entry                                             |
| `span_id_field`    |                       | A [field](/docs/types/field.md) for the span_id on the log entry                                           |
| `use_compression`  | `true`                | Whether to compress the log entry payloads with gzip before sending to Google Cloud                        |
| `timeout`          | 10s                   | A [duration](/docs/types/duration.md) indicating how long to wait for the API to respond before timing out |
| `buffer`           |                       | A [buffer](/docs/types/buffer.md) block indicating how to buffer entries before flushing                   |
| `flusher`          |                       | A [flusher](/docs/types/flusher.md) block configuring flushing behavior                                    |
| `max_entry_size`   | 200k                  | Entries that exceed this value are dropped. See [ByteSize](/docs/types/bytesize.md) for details on allowed values. |
| `max_request_size` | 5mb                   | Constrains requests to this size limit. See [ByteSize](/docs/types/bytesize.md) for details on allowed values. |

If both `credentials` and `credentials_file` are left empty, the agent will attempt to find
[Application Default Credentials](https://cloud.google.com/docs/authentication/production) from the environment.

### Example Configurations

#### Simple configuration

Configuration:
```yaml
- type: google_cloud_output
  project_id: sample_project
  credentials_file: /tmp/credentials.json
```

#### Configuration with non-default buffer and flusher params

Configuration:
```yaml
- type: google_cloud_output
  project_id: sample_project
  credentials_file: /tmp/credentials.json
  buffer:
    type: disk
    path: /tmp/stanza_buffer
  flusher:
    max_concurrent: 8
```
