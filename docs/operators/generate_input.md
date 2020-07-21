## `generate_input` operator

The `generate` generates log entries with a static record. This is useful for testing pipelines, especially when
coupled with the [`rate_limit`](/docs/operators/rate_limit.md) operator.

### Configuration Fields

| Field      | Default  | Description                                                                                      |
| ---        | ---      | ---                                                                                              |
| `id`       | required | A unique identifier for the operator                                                             |
| `output`   | required | The connected operator(s) that will receive all outbound entries                                 |
| `write_to` | $        | A [field](/docs/types/field.md) that will be set to the path of the file the entry was read from |
| `entry`    |          | A [entry](/docs/types/entry.md) log entry to repeatedly generate                                 |
| `count`    | 0        | The number of entries to generate before stopping. A value of 0 indicates unlimited   .          |
| `static`   | `false`  | If true, the timestamp of the entry will remain static after each invocation                     |

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
