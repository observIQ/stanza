## `severity_parser` plugin

The `severity_parser` plugin sets the timestamp on an entry by parsing a value from the record.

### Configuration Fields

| Field         | Default   | Description                                                                                   |
| ---           | ---       | ---                                                                                           |
| `id`          | required  | A unique identifier for the plugin                                                            |
| `output`      | required  | The `id` for the plugin to send parsed entries to                                             |
| `parse_from`  | required  | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                 |
| `preserve`    | false     | Preserve the unparsed value on the record                                                     |
| `on_error`    | `send`    | The behavior of the plugin if it encounters an error. See [on_error](/docs/types/on_error.md) |
| `preset`      | `default` | A predefined set of values that should be interpretted at specific severity levels            |
| `mapping`     |           | A formatted set of values that should be interpretted as severity levels.                     |


### Example Configurations

Several detailed examples are available [here](/docs/types/severity.md).