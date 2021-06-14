## `netflow_v5_input` operator

The `netflow_v5_input` operator recieves Sflow messages from network devices

### Configuration Fields

| Field        | Default             | Description                                                                                   |
| ---          | ---                 | ---                                                                                           |
| `id`         | `netflow_v5_input`  | A unique identifier for the operator                                                          |
| `output`     | Next in pipeline    | The connected operator(s) that will receive all outbound entries                              |
| `address`    | `0.0.0.0`           | The ip address to bind to                                                                     |
| `port`       | required            | The port to bind to                                                                           |
| `workers`    | `1`                 | Number of worker processes spawned by the underlying [Goflow package](https://github.com/cloudflare/goflow)  |

### Example Configuration

Configuration:
```yaml
pipeline:
- type: netflow_v5_input
  port: 2000
- type: stdout
```

### Example Output

```json
{
  "timestamp": "2021-06-14T18:48:21.867900348-04:00",
  "severity": 0,
  "record": {
    "bytes": 406,
    "dstaddr": "0.123.54.200",
    "dstas": 9755,
    "dstnet": 11,
    "dstport": 43362,
    "etype": 2048,
    "nexthop": "121.69.89.49",
    "packets": 111,
    "proto": 6,
    "sampleraddress": "172.17.0.2",
    "sequencenum": 1,
    "srcaddr": "84.62.198.80",
    "srcas": 13734,
    "srcnet": 26,
    "srcport": 64344,
    "timeflowend": 1623710901,
    "timeflowstart": 1623710901,
    "timereceived": 1623710901,
    "type": 2
  }
}
```