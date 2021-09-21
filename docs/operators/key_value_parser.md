## `key_value_parser` operator

The `key_value_parser` operator parses the string-type field selected by `parse_from` into key value pairs. All values are of type string.

### Configuration Fields

| Field         | Default             | Description                                                                                                                                                                                                                              |
| ---           | ---                 | ---                                                                                                                                                                                                                                      |
| `id`          | `key_value_parser`  | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`      | Next in pipeline    | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `parse_from`  | $                   | A [field](/docs/types/field.md) that indicates the field to be parsed into key value pairs                                                                                                                                               |
| `parse_to`    | $                   | A [field](/docs/types/field.md) that indicates the field to be parsed as into key value pairs                                                                                                                                            |
| `preserve_to` |                     | Preserves the unparsed value at the specified [field](/docs/types/field.md)                                                                                                                                                              |
| `on_error`    | `send`              | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`          |                     | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |
| `timestamp`   | `nil`               | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator                                                                                               |
| `severity`    | `nil`               | An optional [severity](/docs/types/severity.md) block which will parse a severity field before passing the entry to the output operator                                                                                                  |


### Example Configurations


#### Parse the field `message` into key value pairs

Configuration:
```yaml
- type: key_value_parser
  parse_from: message
```

<table>
<tr><td> Input body </td> <td> Output body </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "body": {
    "message": "name=stanza"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "body": {
    "name": "stanza"
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` as key value pairs, and parse the timestamp

Configuration:
```yaml
- type: key_value_parser
  parse_from: message
  timestamp:
    parse_from: seconds_since_epoch
    layout_type: epoch
    layout: s
```

<table>
<tr><td> Input body </td> <td> Output body </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "body": {
    "message": "name=stanza seconds_since_epoch=1136214245"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2006-01-02T15:04:05-07:00",
  "body": {
    "name": "stanza"
  }
}
```

</td>
</tr>
</table>