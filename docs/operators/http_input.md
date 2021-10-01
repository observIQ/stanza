## `http_input` operator

The `http_input` operator starts an HTTP server for receiving log messages.

### Configuration Fields

| Field              | Default               | Description                                                                                                |
| ---                | ---                   | ---                                                                                                        |
| `id`               | `http_input`          | A unique identifier for the operator                                                                       |
| `listen_address`   | required              | A listen address of the form `<ip>:<port>`                                        |
| `idle_timeout`     | 60s                   |                                        |
| `read_timeout`     | 20s                   |                                        |
| `write_timeout`    | 20s                   |                                        |
| `max_header_size`  | 1mb                   |                                        |
| `max_body_size`    | 10mb                  |                                        |
| `auth`             |                       | An optional `Auth` configuration (see the Auth configuration section)               |
| `tls`              |                       | An optional `TLS` configuration (see the TLS configuration section)               |


#### Auth Configuration

The `http_input` operator supports authentication, disabled by default.

| Field           | Default        | Description                               |
| ---             | ---            | ---                                       |
| `username`      | `""`           | Basic Auth Username                       |
| `password`      | `""`           | Basic Auth Password                       |
| `token_header`  | `""`           | Token auth header, a header that contains a token that matches one of the configured `tokens`               |
| `tokens`        | `[]`           | An array of token values, used to compare against the value found in the header defined with `token_header` |

#### TLS Configuration

The `http_input` operator supports TLS, disabled by default.

| Field             | Default          | Description                               |
| ---               | ---              | ---                                       |
| `enable`          | `false`          | Boolean value to enable or disable TLS    |
| `certificate`     | `""`             | File path for the X509 certificate chain  |
| `private_key`     | `""`             | File path for the X509 private key        |
| `min_version`     | `1.2`            | Minimum TLS version to accept connections |


### Output

```bash
curl localhost:9090/ \
    -X POST \
    -u stanza:dev \
    -d '{"message":"logging enabled","user":"devel","mode":"test"}'
```
```json
{
  "timestamp": "2021-09-24T14:33:56.653226981-04:00",
  "severity": 0,
  "labels": {
    "net.host.ip": "localhost",
    "net.host.port": "9090",
    "net.peer.ip": "::1",
    "net.peer.port": "56554",
    "protocol": "HTTP",
    "protocol_version": "1.1"
  },
  "record": {
    "mode": "test",
    "user": "devel",
    "message": "logging enabled"
  }
}
```

### Example Configurations

#### Simple configuration

Configuration:
```yaml
- type: http_input
  listen_address: 0.0.0.0:9090
```

#### Advanced Configuration with Basic Auth

Configuration:
```yaml
- type: http_input
  listen_address: 0.0.0.0:9090
  idle_timeout: 10ms
  read_timeout: 10ms
  write_timeout: 10ms
  max_header_size: 5000
  max_body_size: 1mb
  auth:
    username: stanza
    password: dev
```

#### Advanced Configuration with token Auth

Configuration:
```yaml
- type: http_input
  listen_address: 0.0.0.0:9090
  idle_timeout: 10ms
  read_timeout: 10ms
  write_timeout: 10ms
  max_header_size: 5000
  max_body_size: 1mb
  auth:
    token: x-secret-key
    values:
    - "token-a"
    - "token-stage"
```

#### TLS

Configuration:
```yaml
- type: http_input
  listen_address: 0.0.0.0:9090
  tls:
    enable: true
    certificate: ./cert
    private_key: ./key
```