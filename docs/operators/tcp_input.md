## `tcp_input` operator

The `tcp_input` operator listens for logs on one or more TCP connections. The operator assumes that logs are newline separated.

### Configuration Fields

| Field             | Default          | Description                                                                       |
| ---               | ---              | ---                                                                               |
| `id`              | `tcp_input`      | A unique identifier for the operator                                              |
| `output`          | Next in pipeline | The connected operator(s) that will receive all outbound entries                  |
| `max_buffer_size` | `1024kib`        | Maximum size of buffer that may be allocated while reading TCP input              |
| `listen_address`  | required         | A listen address of the form `<ip>:<port>`                                        |
| `tls`             |                  | An optional `TLS` configuration (see the TLS configuration section)               |
| `write_to`        | $                | The record [field](/docs/types/field.md) written to when creating a new log entry |
| `labels`          | {}               | A map of `key: value` labels to add to the entry's labels                         |
| `resource`        | {}               | A map of `key: value` labels to add to the entry's resource                       |
| `add_labels`      | false            | Adds `net.transport`, `net.peer.ip`, `net.peer.port`, `net.host.ip` and `net.host.port` labels |

#### TLS Configuration

The `tcp_input` operator supports TLS, disabled by default.

| Field             | Default          | Description                               |
| ---               | ---              | ---                                       |
| `enable`          | `false`          | Boolean value to enable or disable TLS    |
| `certificate`     |                  | File path for the X509 certificate chain  |
| `private_key`     |                  | File path for the X509 private key        |
| `min_version`     | `1.0`            | Minimum TLS version to accept connections from, defaults [TLS 1.0](https://pkg.go.dev/crypto/tls#Config)


### Example Configurations

#### Simple

Configuration:
```yaml
- type: tcp_input
  listen_address: "0.0.0.0:54525"
```

Send a log:
```bash
$ nc localhost 54525 <<EOF
heredoc> message1
heredoc> message2
heredoc> EOF
```

Generated entries:
```json
{
  "timestamp": "2020-04-30T12:10:17.656726-04:00",
  "record": "message1"
},
{
  "timestamp": "2020-04-30T12:10:17.657143-04:00",
  "record": "message2"
}
```

### Example TLS Configurations

#### Simple TLS

Configuration:
```yaml
pipeline:
- type: tcp_input
  listen_address: 0.0.0.0:5000
  tls:
    enable: true
    certificate: ./cert
    private_key: ./key
- type: stdout
```

Send a log:
```bash
echo sample message | openssl s_client -connect localhost:5000
```

Generated entry:
```json
{
  "timestamp": "2021-08-20T19:53:56.905051345-04:00",
  "severity": 0,
  "record": "sample message"
}
```

#### TLS 1.3

Configuration:
```yaml
pipeline:
- type: tcp_input
  listen_address: 0.0.0.0:5000
  tls:
    enable: true
    certificate: ./cert
    private_key: ./key
    min_version: 1.3
- type: stdout
```

Send a log with the `-tls1_3` flag:
```bash
echo sample message | openssl s_client -tls1_3 -connect localhost:5000
```

Generated entry:
```json
{
  "timestamp": "2021-08-20T19:53:56.905051345-04:00",
  "severity": 0,
  "record": "sample message"
}
```

Try it a second time using a lower TLS version, such as `-tls1_2`, and it will fail:
```json
{
  "level":"error",
  "timestamp":"2021-08-20T19:56:38.108-0400",
  "message":"Scanner error",
  "operator_id":"$.tcp_input",
  "operator_type":"tcp_input",
  "error":"tls: client offered only unsupported versions: [303 302 301]"
}
```