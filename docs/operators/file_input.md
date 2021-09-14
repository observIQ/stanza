## `file_input` operator

The `file_input` operator reads logs from files. It will place the lines read into the `message` field of the new entry.

### Configuration Fields

| Field                  | Default          | Description                                                                                                        |
| ---                    | ---              | ---                                                                                                                |
| `id`                   | `file_input`     | A unique identifier for the operator                                                                               |
| `output`               | Next in pipeline | The connected operator(s) that will receive all outbound entries                                                   |
| `include`              | required         | A list of file glob patterns that match the file paths to be read                                                  |
| `exclude`              | []               | A list of file glob patterns to exclude from reading                                                               |
| `poll_interval`        | 200ms            | The duration between filesystem polls                                                                              |
| `multiline`            |                  | A `multiline` configuration block. See below for details                                                           |
| `write_to`             | $                | The record [field](/docs/types/field.md) written to when creating a new log entry                                  |
| `encoding`             | `nop`            | The encoding of the file being read. See the list of supported encodings below for available options               |
| `include_file_name`    | `true`           | Whether to add the file name as the label `file_name`                                                              |
| `include_file_path`    | `false`          | Whether to add the file path as the label `file_path`                                                              |
| `include_file_name_resolved`    | `false`          | Whether to add the file name after symlinks resolution as the label `file_name_resolved`                  |
| `include_file_path_resolved`    | `false`          | Whether to add the file path after symlinks resolution as the label `file_path_resolved`                  |
| `start_at`             | `end`            | At startup, where to start reading logs from the file. Options are `beginning` or `end`                            |
| `delete_after_read`    | `false`          | After reading a to the end of a file, delete it. Cannot be `true` when `start_at` is `end`.                        |
| `fingerprint_size`     | `1kb`            | The number of bytes with which to identify a file. The first bytes in the file are used as the fingerprint. Decreasing this value at any point will cause existing fingerprints to forgotten, meaning that all files will be read from the beginning (one time). |
| `max_log_size`         | `1MiB`           | The maximum size of a log entry to read before failing. Protects against reading large amounts of data into memory |
| `max_concurrent_files` | 1024             | The maximum number of log files from which logs will be read concurrently (minimum = 2). If the number of files matched in the `include` pattern exceeds half of this number, then files will be processed in batches. One batch will be processed per `poll_interval`. |
| `labels`               | {}               | A map of `key: value` labels to add to the entry's labels                                                          |
| `resource`             | {}               | A map of `key: value` labels to add to the entry's resource                                                        |

Note that by default, no logs will be read unless the monitored file is actively being written to because `start_at` defaults to `end`.

`include` and `exclude` fields use `github.com/bmatcuk/doublestar` for expression language.
For reference documentation see [here](https://github.com/bmatcuk/doublestar#patterns).

#### `multiline` configuration

If set, the `multiline` configuration block instructs the `file_input` operator to split log entries on a pattern other than newlines.

The `multiline` configuration block must contain exactly one of `line_start_pattern` or `line_end_pattern`. These are regex patterns that
match either the beginning of a new log entry, or the end of a log entry.

Also refer to [recombine](/docs/operators/recombine.md) operator for merging events with greater control. 

### File rotation

When files are rotated and its new names are no longer captured in `include` pattern (i.e. tailing symlink files), it could result in data loss.
To avoid the data loss, choose move/create rotation method and set `max_concurrent_files` higher than the twice of the number of files to tail. 

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
