## `csv_parser` operator

The `csv_parser` operator parses the string-type field selected by `parse_from` with the given header values.

### Configuration Fields

| Field          | Default          | Description                                                                                                                                                                                                                              |
| ---            | ---              | ---                                                                                                                                                                                                                                      |
| `id`           | `csv_parser`     | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`       | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `header`       | required when `header_attribute` not set  | A string of delimited field names                                                                                                                            |
| `header_attribute` | required when `header` not set        | A attribute name to read the header field from, to support dynamic field names                                                                                                                                          |
| `header_delimiter`   | value of delimiter              | A character that will be used as a delimiter for the header. Values `\r` and `\n` cannot be used as a delimiter                                                                                                     |
| `delimiter`    | `,`              | A character that will be used as a delimiter. Values `\r` and `\n` cannot be used as a delimiter                                                                                                                                         |
| `parse_from`   | $                | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                                                                                                                    |
| `parse_to`     | $                | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                                                                                                                    |
| `preserve_to`  |                  | Preserves the unparsed value at the specified [field](/docs/types/field.md)                                                                                                                                                              |
| `on_error`     | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `timestamp`    | `nil`            | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator                                                                                               |
| `severity`     | `nil`            | An optional [severity](/docs/types/severity.md) block which will parse a severity field before passing the entry to the output operator                                                                                                  |

### Example Configurations

#### Parse the field `message` with a csv parser

Configuration:

```yaml
- type: csv_parser
  parse_from: message
  header: 'id,severity,message'
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "1,debug,\"\"Debug Message\"\""
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "id": "1",
    "severity": "debug",
    "message": "\"Debug Message\""
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` with a csv parser using tab delimiter

Configuration:

```yaml
- type: csv_parser
  parse_from: message
  header: 'id,severity,message'
  header_delimiter: ","
  delimiter: "\t"
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "1 debug \"Debug Message\""
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "id": "1",
    "severity": "debug",
    "message": "\"Debug Message\""
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` with csv parser and also parse the timestamp

Configuration:

```yaml
- type: csv_parser
  header: 'timestamp_field,severity,message'
  timestamp:
    parse_from: timestamp_field
    layout_type: strptime
    layout: '%Y-%m-%d'
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "2021-03-17,debug,Debug Message"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2021-03-17T00:00:00-00:00",
  "record": {
    "severity": "debug",
    "message": "Debug Message"
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` with differing delimiters for header and fields

Configuration:

```yaml
- type: csv_parser
  parse_from: message
  delimiter: "+"
  header_delimiter: ","
  header: 'id,severity,message'
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": {
    "message": "1+debug+\"\"Debug Message\"\""
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "id": "1",
    "severity": "debug",
    "message": "\"Debug Message\""
  }
}
```

</td>
</tr>
</table>

#### Parse the field `message` using dynamic field names

Dynamic field names can be had when leveraging file_input's `attribute_regex`.

Configuration:

```yaml
- type: file_input
  include:
  - ./dynamic.log
  start_at: beginning
  attribute_regex: '^#(?P<key>.*?): (?P<value>.*)'

- type: csv_parser
  delimiter: ","
  header_attribute: Fields
```

Input File:

```
#Fields: "id,severity,message"
1,debug,Hello
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

Entry (from file_input):

```json
{
  "timestamp": "",
  "attributes": {
    "fields": "id,severity,message"
  },
  "record": {
    "message": "1,debug,Hello"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "attributes": {
    "fields": "id,severity,message"
  },
  "record": {
    "id": "1",
    "severity": "debug",
    "message": "Hello"
  }
}
```

</td>
</tr>
</table>