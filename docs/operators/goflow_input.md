## `goflow_input` operator

The `goflow_input` operator recieves Netflow v9, Netflow v5, and Sflow messages from network devices. `goflow_input` implements [Goflow](https://github.com/cloudflare/goflow).

### Configuration Fields

| Field        | Default             | Description                                                                                   |
| ---          | ---                 | ---                                                                                           |
| `id`         | `goflow_input`      | A unique identifier for the operator                                                          |
| `output`     | Next in pipeline    | The connected operator(s) that will receive all outbound entries                              |
| `mode`       | required            | The Goflow mode [`netflow_v9`, `netflow_v5`, `sflow`]                                         |
| `address`    | `0.0.0.0`           | The ip address to bind to                                                                     |
| `port`       | required            | The port to bind to                                                                           |
| `workers`    | `1`                 | Number of worker processes spawned by the underlying [Goflow package](https://github.com/cloudflare/goflow)  |

### Example Configuration

Configuration:
```yaml
pipeline:
- type: goflow_input
  mode: netflow_v5
  port: 2000
- type: stdout
```