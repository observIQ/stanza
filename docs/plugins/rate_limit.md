## `rate_limit` plugin

The `rate_limit` limits the rate of entries that can pass through it. This is useful if you want to limit
throughput of the log agent, or in conjunction with plugins like `generate_input`, which will otherwise
send as fast as possible.

### Configuration Fields

| Field      | Default  | Description                                                                        |
| ---        | ---      | ---                                                                                |
| `id`       | required | A unique identifier for the plugin                                                 |
| `output`   | required | The connected plugin(s) that will receive all outbound entries                     |
| `rate`     |          | The number of logs to allow per second                                             |
| `interval` |          | A [duration](/docs/types/duration.md) that indicates the time between sent entries |
| `burst`    | 0        | The max number of entries to "save up" for spikes of load                          |

Exactly one of `rate` or `interval` must be specified.

### Example Configurations


#### Limit throughput to 10 entries per second

Configuration:
```yaml
- id: my_rate_limiter
  type: rate_limit
  rate: 10
  output: rate_limit_receiver
```
