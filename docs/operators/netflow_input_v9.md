## `netflow_v9_input` operator

The `netflow_v9_input` operator recieves Sflow messages from network devices

### Configuration Fields

| Field        | Default             | Description                                                                                   |
| ---          | ---                 | ---                                                                                           |
| `id`         | `netflow_v9_input`  | A unique identifier for the operator                                                          |
| `output`     | Next in pipeline    | The connected operator(s) that will receive all outbound entries                              |
| `address`    | `0.0.0.0`           | The ip address to bind to                                                                     |
| `port`       | required            | The port to bind to                                                                           |
| `workers`    | `1`                 | Number of worker processes spawned by the underlying [Goflow package](https://github.com/cloudflare/goflow)  |

### Example Configuration

Configuration:
```yaml
pipeline:
- type: netflow_v9_input
  port: 2000
- type: stdout
```