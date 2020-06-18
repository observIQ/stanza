## `generate_input` plugin

The `generate` generates log entries with a static record. This is useful for testing pipelines, especially when
coupled with the [`rate_limit`](/docs/plugins/rate_limit.md) plugin.

### Configuration Fields

| Field      | Default  | Description                                                                                      |
| ---        | ---      | ---                                                                                              |
| `id`       | required | A unique identifier for the plugin                                                               |
| `output`   | required | The `id` for the plugin to send parsed entries to                                                |
| `write_to` | $        | A [field](/docs/types/field.md) that will be set to the path of the file the entry was read from |
| `record`   |          | A log entry record to repeatedly generate. Must be either a string or map                        |
| `count`    | 0        | The number of entries to generate before stopping. A value of 0 indicates unlimited              |


### Example Configurations

#### Mock a file input

Configuration:
```yaml
- id: my_generate_input
  type: generate_input
  output: generate_input_receiver
  record:
    message1: log1
    message2: log2
```

Output records:
```json
{
  "message1": "log1",
  "message2": "log2"
},
{
  "message1": "log1",
  "message2": "log2"
},
...
```
