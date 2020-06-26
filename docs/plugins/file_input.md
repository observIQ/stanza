## `file_input` plugin

The `file_input` plugin reads logs from files. It will place the lines read into the `message` field of the new entry.

### Configuration Fields

| Field           | Default     | Description                                                                                                         |
| ---             | ---         | ---                                                                                                                 |
| `id`            | required    | A unique identifier for the plugin                                                                                  |
| `output`        | required    | The connected plugin(s) that will receive all outbound entries                                                      |
| `include`       | required    | A list of file glob patterns that match the file paths to be read                                                   |
| `exclude`       | []          | A list of file glob patterns to exclude from reading                                                                |
| `poll_interval` | 200ms       | The duration between filesystem polls                                                                               |
| `multiline`     |             | A `multiline` configuration block. See below for details                                                            |
| `write_to`      | $           | A [field](/docs/types/field.md) that will be set to the path of the file the entry was read from                    |
| `path_field`    |             | A [field](/docs/types/field.md) that will be set to the path of the file the entry was read from                    |
| `start_at`      | `beginning` | At startup, where to start reading logs from the file. Options are `beginning` or `end`                             |
| `max_log_size`  | 1048576     | The maximum size of a log entry to read before failing. Protects against reading large amounts of data into memory. |

#### `multiline` configuration

If set, the `multiline` configuration block instructs the `file_input` plugin to split log entries on a pattern other than newlines.

The `multiline` configuration block must contain exactly one of `line_start_pattern` or `line_end_pattern`. These are regex patterns that
match either the beginning of a new log entry, or the end of a log entry.

### Example Configurations

#### Simple file input

Configuration:
```yaml
- id: my_file_input
  type: file_input
  include:
    - ./test.log
  output: file_input_receiver
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
- id: my_file_input
  type: file_input
  include:
    - ./test.log
  multiline:
    line_start_pattern: 'START '
  output: file_input_receiver
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
