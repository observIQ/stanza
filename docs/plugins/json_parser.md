## `json_parser` plugin

The `json_parser` plugin parses the string-type field selected by `parse_from` as JSON.

### Configuration Fields

| Field        | Default  | Description                                                                                                                                     |
| ---          | ---      | ---                                                                                                                                             |
| `id`         | required | A unique identifier for the plugin                                                                                                              |
| `output`     | required | The `id` for the plugin to send parsed entries to                                                                                               |
| `parse_from` | $        | A [field](/docs/field.md) that indicates the field to be parsed as JSON                                                                         |
| `parse_to`   | $        | A [field](/docs/field.md) that indicates the field to be parsed as JSON                                                                         |
| `preserve`   | false    | Preserve the unparsed value on the record                                                                                                       |
| `on_error`   | "ignore" | The behavior of the plugin if it encounters an error. See [on_error](/TODO)                                                                     |
| `timestamp`  | `nil`    | An optional [timestamp](/docs/timestamp.md) block which will parse a timestamp field before passing the entry to the output plugin              |


### Example Configurations


#### Parse the field `message` as JSON

Configuration:
```yaml
- id: my_json_parser
  type: json_parser
  parse_from: message
  output: parsed_json_receiver
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "{\"key\": \"val\"}"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "key": "val"
  }
}
```

</td>
</tr>
</table>

#### Parse a nested field to a different field, preserving original

Configuration:
```yaml
- id: my_json_parser
  type: json_parser
  parse_from: message.embedded
  parse_to: parsed
  preserve: true
  output: parsed_json_receiver
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
      "embedded": "{\"key\": \"val\"}"
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
      "embedded": "{\"key\": \"val\"}"
    },
    "parsed": {
      "key": "val"
    }
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` as JSON, and parse the timestamp

Configuration:
```yaml
- id: my_json_parser
  type: json_parser
  parse_from: message
  timestamp:
    parse_from: seconds_since_epoch
    layout_type: epoch
    layout: s
  output: parsed_json_receiver
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "{\"key\": \"val\", \"seconds_since_epoch\": 1136214245}"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2006-01-02T15:04:05-07:00",
  "record": {
    "key": "val"
  }
}
```

</td>
</tr>
</table>