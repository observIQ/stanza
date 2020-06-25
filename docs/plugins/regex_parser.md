## `regex_parser` plugin

The `regex` plugin parses the string-type field selected by `parse_from` with the given regular expression pattern.

### Configuration Fields

| Field        | Default  | Description                                                                                                                                     |
| ---          | ---      | ---                                                                                                                                             |
| `id`         | required | A unique identifier for the plugin                                                                                                              |
| `output`     | required | The connected plugin(s) that will receive all outbound entries                                                                               |
| `regex`      | required | A [Go regular expression](https://github.com/google/re2/wiki/Syntax). The named capture groups will be extracted as fields in the parsed object |
| `parse_from` | $        | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                           |
| `parse_to`   | $        | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                           |
| `preserve`   | false    | Preserve the unparsed value on the record                                                                                                       |
| `on_error`   | `send` | The behavior of the plugin if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                     |
| `timestamp`  | `nil`    | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output plugin        |

### Example Configurations


#### Parse the field `message` with a regular expression

Configuration:
```yaml
- id: my_regex_parser
  type: regex_parser
  parse_from: message
  regexp: '^Host=(?P<host>[^,]+), Type=(?P<type>.*)$'
  output: parsed_regex_receiver
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "Host=127.0.0.1, Type=HTTP"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "host": "127.0.0.1",
    "type": "HTTP"
  }
}
```

</td>
</tr>
</table>

#### Parse a nested field to a different field, preserving original

Configuration:
```yaml
- id: my_regex_parser
  type: regex_parser
  parse_from: message.embedded
  parse_to: parsed
  preserve: true
  output: parsed_regex_receiver
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": {
      "embedded": "Host=127.0.0.1, Type=HTTP"
    }
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": {
      "embedded": "Host=127.0.0.1, Type=HTTP"
    },
    "parsed": {
      "host": "127.0.0.1",
      "type": "HTTP"
    }
  }
}
```

</td>
</tr>
</table>


#### Parse the field `message` with a regular expression and also parse the timestamp

Configuration:
```yaml
- id: my_regex_parser
  type: regex_parser
  regexp: '^Time=(?P<timestamp_field>\d{4}-\d{2}-\d{2}), Host=(?P<host>[^,]+)'
  timestamp:
    parse_from: timestamp_field
    layout_type: strptime
    layout: '%Y-%m-%d'
  output: my_next_plugin
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "Time=2020-01-31, Host=127.0.0.1, Type=HTTP"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-01-31T00:00:00-00:00",
  "record": {
    "host": "127.0.0.1",
    "type": "HTTP"
  }
}
```

</td>
</tr>
</table>




