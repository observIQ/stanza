## `udp_input` plugin

The `udp_input` plugin listens for logs from UDP packets.

### Configuration Fields

| Field            | Default  | Description                                                         |
| ---              | ---      | ---                                                                 |
| `id`             | required | A unique identifier for the plugin                                  |
| `output`         | required | The `id` for the plugin to send parsed entries to                   |
| `listen_address` | required | A listen address of the form `<ip>:<port>`                          |
| `write_to`       | $        | A [field](/docs/types/field.md) that will be set to the log message |

### Example Configurations

#### Simple

Configuration:
```yaml
- id: my_udp_input
  type: udp_input
  listen_adress: "0.0.0.0:54526"
  output: udp_receiver
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
