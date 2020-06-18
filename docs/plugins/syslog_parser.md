## `syslog_parser` plugin

The `syslog_parser` plugin parses the string-type field selected by `parse_from` as syslog. Timestamp parsing is handled automatically by this plugin.

### Configuration Fields

| Field        | Default  | Description                                                                                  |
| ---          | ---      | ---                                                                                          |
| `id`         | required | A unique identifier for the plugin                                                           |
| `output`     | required | The `id` for the plugin to send parsed entries to                                            |
| `parse_from` | $        | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                |
| `parse_to`   | $        | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                |
| `preserve`   | false    | Preserve the unparsed value on the record                                                    |
| `on_error`   | "ignore" | The behavior of the plugin if it encounters an error. See [on_error](/TODO)                  |
| `protocol`   | required | The protocol to parse the syslog messages as. Options are `rfc3164` and `rfc5424`            |

### Example Configurations


#### Parse the field `message` as syslog

Configuration:
```yaml
- id: my_syslog_parser
  type: syslog_parser
  protocol: rfc3164
  output: parsed_syslog_receiver
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": "<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message"
}
```

</td>
<td>

```json
{
  "timestamp": "2020-01-12T06:30:00Z",
  "record": {
    "appname": "apache_server",
    "facility": 4,
    "hostname": "1.2.3.4",
    "message": "test message",
    "msg_id": null,
    "priority": 34,
    "proc_id": null,
    "severity": 2,
  }
}
```

</td>
</tr>
</table>
