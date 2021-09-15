## `goflow_input` operator

The `goflow_input` operator recieves Netflow v9 / IPFIX, Netflow v5, and Sflow messages from network devices. `goflow_input` implements [Goflow](https://github.com/cloudflare/goflow).

The `timereceived` field is promoted as the entries Timestamp.

### Configuration Fields

| Field        | Default             | Description                                                                                   |
| ---          | ---                 | ---                                                                                           |
| `id`         | `goflow_input`      | A unique identifier for the operator                                                          |
| `output`     | Next in pipeline    | The connected operator(s) that will receive all outbound entries                              |
| `mode`       | `netflow_ipfix`     | The Goflow mode [`netflow_ipfix`, `netflow_v5`, `sflow`]                                      |
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

### Example Output

```json
{
  "timestamp": "2021-06-15T11:59:26-04:00",
  "severity": 0,
  "body": {
    "bytes": 936,
    "dstaddr": "173.195.121.172",
    "dstas": 14164,
    "dstnet": 5,
    "dstport": 17210,
    "etype": 2048,
    "nexthop": "66.88.34.2",
    "packets": 100,
    "proto": 6,
    "sampleraddress": "172.17.0.2",
    "sequencenum": 7,
    "srcaddr": "241.104.80.243",
    "srcas": 43137,
    "srcnet": 11,
    "srcport": 37247,
    "timeflowend": 1623772766,
    "timeflowstart": 1623772766,
    "type": 2
  }
}

```