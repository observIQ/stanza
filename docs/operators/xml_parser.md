## `xml_parser` operator

The `xml_parser` operator parses the string-type field selected by `parse_from` as XML.

### Configuration Fields

| Field         | Default          | Description                                                                                                                                                                                                                              |
| ---           | ---              | ---                                                                                                                                                                                                                                      |
| `id`          | `xml_parser`    | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`      | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `parse_from`  | $                | A [field](/docs/types/field.md) that indicates the field to be parsed as XML                                                                                                                                                            |
| `parse_to`    | $                | A [field](/docs/types/field.md) that indicates the field to be parsed as XML                                                                                                                                                            |
| `preserve_to` |                  | Preserves the unparsed value at the specified [field](/docs/types/field.md)                                                                                                                                                              |
| `on_error`    | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`          |                  | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |
| `timestamp`   | `nil`            | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator                                                                                               |
| `severity`    | `nil`            | An optional [severity](/docs/types/severity.md) block which will parse a severity field before passing the entry to the output operator                                                                                                  |


### Example Configurations


#### Parse the field `message` as XML

Configuration:
```yaml
- type: xml_parser
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
    "message": "<person age='30'>Jon Smith</person>"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "type": "person",
    "attributes": {
      "age": "30"
    },
    "value": "Jon Smith"
  }
}
```

</td>
</tr>
</table>

#### Parse multiple xml elements

Configuration:
```yaml
- type: xml_parser
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
    "message": "<person age='30'>Jon Smith</person><person age='28'>Sally Smith</person>"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": [
    {
    "type": "person",
    "attributes": {
      "age": "30"
    },
    "value": "Jon Smith"
    },
    {
    "type": "person",
    "attributes": {
      "age": "28"
    },
    "value": "Sally Smith"
    }
  ]
}
```

#### Parse embedded xml elements

Configuration:
```yaml
- type: xml_parser
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
    "message": "<worker><person age='30'>Jon Smith</person></worker>"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "type": "worker",
    "children": [
      {
        "type": "person",
        "attributes": {
          "age": "30"
        },
        "value": "Jon Smith"
      }
    ]
  }
}
```
