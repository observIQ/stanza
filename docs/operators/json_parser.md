## `json_parser` operator

The `json_parser` operator parses the string-type field selected by `parse_from` as JSON.

### Configuration Fields

| Field        | Default          | Description                                                                                                                                |
| ---          | ---              | ---                                                                                                                                        |
| `id`         | `json_parser`    | A unique identifier for the operator                                                                                                       |
| `output`     | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                           |
| `parse_from` | $                | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                                                              |
| `parse_to`   | $                | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                                                              |
| `preserve`   | false            | Preserve the unparsed value on the record                                                                                                  |
| `on_error`   | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                            |
| `timestamp`  | `nil`            | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator |


### Example Configurations


#### Parse the field `message` as JSON

Configuration:
```yaml
- type: json_parser
  parse_from: message
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
- type: json_parser
  parse_from: message.embedded
  parse_to: parsed
  preserve: true
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
- type: json_parser
  parse_from: message
  timestamp:
    parse_from: seconds_since_epoch
    layout_type: epoch
    layout: s
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
