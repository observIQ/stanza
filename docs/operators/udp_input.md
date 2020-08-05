## `udp_input` operator

The `udp_input` operator listens for logs from UDP packets.

### Configuration Fields

| Field             | Default          | Description                                                         |
| ---               | ---              | ---                                                                 |
| `id`              | `udp_input`      | A unique identifier for the operator                                |
| `output`          | Next in pipeline | The connected operator(s) that will receive all outbound entries    |
| `listen_address`  | required         | A listen address of the form `<ip>:<port>`                          |
| `write_to`        | $                | A [field](/docs/types/field.md) that will be set to the log message |
| `log_type`        | `udp_input`      | The log_type label appended to all discovered entries               |
| `append_log_type` | `true`           | If true, appends the log_type label to all entries                  |

### Example Configurations

#### Simple

Configuration:
```yaml
- type: udp_input
  listen_adress: "0.0.0.0:54526"
```

Send a log:
```bash
$ nc -u localhost 54525 <<EOF
heredoc> message1
heredoc> message2
heredoc> EOF
```

Generated entries:
```json
{
  "timestamp": "2020-04-30T12:10:17.656726-04:00",
  "record": "message1\nmessage2\n"
}
```
