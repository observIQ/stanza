## `regex_parser` plugin

The `regex` plugin parses the string-type field selected by `parse_from` with the given regular expression pattern.

### Configuration Fields

| Field        | Default  | Description                                                                                                                                          |
| ---          | ---      | ---                                                                                                                                                  |
| `id`         | required | A unique identifier for the plugin                                                                                                                   |
| `output`     | required | The `id` for the plugin to send parsed entries to                                                                                                    |
| `regex`      | required | A [Go regular expression](https://github.com/google/re2/wiki/Syntax). The named capture groups will be extracted as fields in the parsed object      |
| `parse_from` | $        | A [field](/docs/field.md) that indicates the field to be parsed                                                                                      |
| `parse_to`   | $        | A [field](/docs/field.md) that indicates the field to be parsed                                                                                      |
| `preserve`   | false    | Preserve the unparsed value on the record                                                                                                            |
| `on_error`   | "ignore" | The behavior of the plugin if it encounters an error. See [on_error](/TODO)                                                                          |

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
  "message": "Host=127.0.0.1, Type=HTTP"
}
```

</td>
<td>

```json
{
  "host": "127.0.0.1",
  "type": "HTTP"
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
  "message": {
    "embedded": "Host=127.0.0.1, Type=HTTP"
  }
}
```

</td>
<td>

```json
{
  "message": {
    "embedded": "Host=127.0.0.1, Type=HTTP"
  },
  "parsed": {
    "host": "127.0.0.1",
    "type": "HTTP"
  }
}
```

</td>
</tr>
</table>
