## `tcp_input` plugin

The `tcp_input` plugin listens for logs on one or more TCP connections. The plugin assumes that logs are newline separated.

### Configuration Fields

| Field            | Default          | Description                                                         |
| ---              | ---              | ---                                                                 |
| `id`             | `tcp_input`      | A unique identifier for the plugin                                  |
| `output`         | Next in pipeline | The connected plugin(s) that will receive all outbound entries      |
| `listen_address` | required         | A listen address of the form `<ip>:<port>`                          |
| `write_to`       | $                | A [field](/docs/types/field.md) that will be set to the log message |

### Example Configurations

#### Simple

Configuration:
```yaml
- type: tcp_input
  listen_adress: "0.0.0.0:54525"
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
