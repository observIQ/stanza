## `time_parser` plugin

The `time_parser` plugin sets the timestamp on an entry by parsing a value from the record.

### Configuration Fields

| Field         | Default    | Description                                                                      |
| ---           | ---        | ---                                                                              |
| `id`          | required   | A unique identifier for the plugin                                               |
| `output`      | required   | The `id` for the plugin to send parsed entries to                                |
| `parse_from`  | required   | A [field](/docs/field.md) that indicates the field to be parsed as JSON          |
| `layout_type` | `strptime` | The type of timestamp. Valid values are `strptime`, `gotime`, and `epoch`        |
| `layout`      | required   | The exact layout of the timestamp to be parsed                                   |
| `preserve`    | false      | Preserve the unparsed value on the record                                        |

### Example Configurations

Several detailed examples are available [here](/docs/timestamp.md).