## `regex_parser` operator

The `regex_parser` operator parses the string-type field selected by `parse_from` with the given regular expression pattern.

### Configuration Fields

| Field         | Default          | Description                                                                                                                                                                                                                              |
| ---           | ---              | ---                                                                                                                                                                                                                                      |
| `id`          | `regex_parser`   | A unique identifier for the operator                                                                                                                                                                                                     |
| `output`      | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                                                                                                                                         |
| `regex`       | required         | A [Go regular expression](https://github.com/google/re2/wiki/Syntax). The named capture groups will be extracted as fields in the parsed object                                                                                          |
| `parse_from`  | $                | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                                                                                                                    |
| `parse_to`    | $                | A [field](/docs/types/field.md) that indicates the field to be parsed                                                                                                                                                                    |
| `preserve_to` |                  | Preserves the unparsed value at the specified [field](/docs/types/field.md)                                                                                                                                                              |
| `on_error`    | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                          |
| `if`          |                  | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |
| `timestamp`   | `nil`            | An optional [timestamp](/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator                                                                                               |
| `severity`    | `nil`            | An optional [severity](/docs/types/severity.md) block which will parse a severity field before passing the entry to the output operator                                                                                                  |
| `cache`       | `nil`            | An optional `cache` block which will cache `regex_parser`'s output, see the `cache` configuration section.                                                                                                                               |

### Cache

Regex is an expensive operation, caching is useful when parsing the same value repeatedly. For example, extracting fields derived from a kubernetes pod's log file name.
Enabling caching can have a negative effect on memory, therefore it is a good idea to limit the cache size to something realistic.

| Field         | Default          | Description                                                     |
| ---           | ---              | ---                                                             |
| `type`        | `memory`         | Cache type to use. Currently, only `memory` cache is supported. |
| `size`        | `0`              | Max number of items to keep in the cache. `0` will allow unlimited size. The cache will delete the oldest item to make room for a new item when max capacity is reached. |

### Example Configurations


#### Parse the field `message` with a regular expression

Configuration:
```yaml
- type: regex_parser
  parse_from: message
  regex: '^Host=(?P<host>[^,]+), Type=(?P<type>.*)$'
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

#### Parse a file name with a regular expression and cache the result

Configuration:
```yaml
- type: regex_parser
  regex: '^(?P<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?P<namespace>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$'
  cache:
    type: memory
    size: 5
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "timestamp": "",
  "record": "coredns-5644d7b6d9-mzngq_kube-system_coredns-901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6.log"
}
```

</td>
<td>

```json
{
  "timestamp": "",
  "record": {
    "container_id": "901f7510281180a402936c92f5bc0f3557f5a21ccb5a4591c5bf98f3ddbffdd6",
    "container_name": "coredns",
    "namespace": "kube-system",
    "pod_name": "coredns-5644d7b6d9-mzngq"
  }
}
```

</td>
</tr>
</table>

#### Parse a nested field to a different field, preserving original

Configuration:
```yaml
- type: regex_parser
  parse_from: message.embedded
  parse_to: parsed
  regex: '^Host=(?P<host>[^,]+), Type=(?P<type>.*)$'
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
- type: regex_parser
  regex: '^Time=(?P<timestamp_field>\d{4}-\d{2}-\d{2}), Host=(?P<host>[^,]+), Type=(?P<type>.*)$'
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

#### Parse the message field only if "type" is "hostname"

Configuration:
```yaml
- type: regex_parser
  regex: '^Host=(?<host>)$'
  if: '$record.type == "hostname"'
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "record": {
    "message": "Host=testhost",
    "type": "hostname"
  }
}
```

</td>
<td>

```json
{
  "record": {
    "host": "testhost",
    "type": "hostname"
  }
}
```

</td>
</tr>

<tr>
<td>

```json
{
  "record": {
    "message": "Key=value",
    "type": "keypair"
  }
}
```

</td>
<td>

```json
{
  "record": {
    "message": "Key=value",
    "type": "keypair"
  }
}
```

</td>
</tr>
</table>

#### Parse the message field only if "type" is "hostname"

Configuration:
```yaml
- type: regex_parser
  regex: '^Host=(?<host>)$'
  if: '$record.type == "hostname"'
```

<table>
<tr><td> Input record </td> <td> Output record </td></tr>
<tr>
<td>

```json
{
  "record": {
    "message": "Host=testhost",
    "type": "hostname"
  }
}
```

</td>
<td>

```json
{
  "record": {
    "host": "testhost",
    "type": "hostname"
  }
}
```

</td>
</tr>

<tr>
<td>

```json
{
  "record": {
    "message": "Key=value",
    "type": "keypair"
  }
}
```

</td>
<td>

```json
{
  "record": {
    "message": "Key=value",
    "type": "keypair"
  }
}
```

</td>
</tr>
</table>
