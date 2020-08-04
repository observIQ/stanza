## `file_input` operator

The `file_input` operator reads logs from files. It will place the lines read into the `message` field of the new entry.

### Configuration Fields

| Field             | Default          | Description                                                                                                         |
| ---               | ---              | ---                                                                                                                 |
| `id`              | `file_input`     | A unique identifier for the operator                                                                                |
| `output`          | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                    |
| `include`         | required         | A list of file glob patterns that match the file paths to be read                                                   |
| `exclude`         | []               | A list of file glob patterns to exclude from reading                                                                |
| `poll_interval`   | 200ms            | The duration between filesystem polls                                                                               |
| `multiline`       |                  | A `multiline` configuration block. See below for details                                                            |
| `write_to`        | $                | A [field](/docs/types/field.md) that will be set to the log message                                                 |
| `encoding`        | `nop`            | The encoding of the file being read. See the list of supported encodings below for available options                |
| `file_path_field` |                  | A [field](/docs/types/field.md) that will be set to the path of the file the entry was read from                    |
| `file_name_field` |                  | A [field](/docs/types/field.md) that will be set to the name of the file the entry was read from                    |
| `start_at`        | `end`            | At startup, where to start reading logs from the file. Options are `beginning` or `end`                             |
| `max_log_size`    | 1048576          | The maximum size of a log entry to read before failing. Protects against reading large amounts of data into memory. |
| `labels`          | {}               | A map of `key: value` labels to add to the entry                                                                    |

Note that by default, no logs will be read unless the monitored file is actively being written to because `start_at` defaults to `end`.

#### `multiline` configuration

If set, the `multiline` configuration block instructs the `file_input` operator to split log entries on a pattern other than newlines.

The `multiline` configuration block must contain exactly one of `line_start_pattern` or `line_end_pattern`. These are regex patterns that
match either the beginning of a new log entry, or the end of a log entry.

### Supported encodings

| Key        | Description
| ---        | ---                                                              |
| `nop`      | No encoding validation. Treats the file as a stream of raw bytes |
| `utf-8`    | UTF-8 encoding                                                   |
| `utf-16le` | UTF-16 encoding with little-endian byte order                    |
| `utf-16be` | UTF-16 encoding with little-endian byte order                    |
| `ascii`    | ASCII encoding                                                   |
| `big5`     | The Big5 Chinese character encoding                              |

Other less common encodings are supported on a best-effort basis. See [https://www.iana.org/assignments/character-sets/character-sets.xhtml](https://www.iana.org/assignments/character-sets/character-sets.xhtml) for other encodings available.


### Example Configurations

#### Simple file input

Configuration:
```yaml
- type: file_input
  include:
    - ./test.log
```

<table>
<tr><td> `./test.log` </td> <td> Output records </td></tr>
<tr>
<td>

```
log1
log2
log3
```

</td>
<td>

```json
{
  "message": "log1"
},
{
  "message": "log2"
},
{
  "message": "log3"
}
```

</td>
</tr>
</table>

#### Multiline file input

Configuration:
```yaml
- type: file_input
  include:
    - ./test.log
  multiline:
    line_start_pattern: 'START '
```

<table>
<tr><td> `./test.log` </td> <td> Output records </td></tr>
<tr>
<td>

```
START log1
log2
START log3
log4
```

</td>
<td>

```json
{
  "message": "START log1\nlog2\n"
},
{
  "message": "START log3\nlog4\n"
}
```

</td>
</tr>
</table>
