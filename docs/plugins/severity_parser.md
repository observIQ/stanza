## `severity_parser` plugin

The `severity_parser` plugin sets the severity on an entry by parsing a value from the record.

### Configuration Fields

| Field        | Default           | Description                                                                                   |
| ---          | ---               | ---                                                                                           |
| `id`         | `severity_parser` | A unique identifier for the plugin                                                            |
| `output`     | Next in pipeline  | The connected plugin(s) that will receive all outbound entries                                |
| `parse_from` | required          | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                 |
| `preserve`   | false             | Preserve the unparsed value on the record                                                     |
| `on_error`   | `send`            | The behavior of the plugin if it encounters an error. See [on_error](/docs/types/on_error.md) |
| `preset`     | `default`         | A predefined set of values that should be interpreted at specific severity levels             |
| `mapping`    |                   | A formatted set of values that should be interpreted as severity levels.                      |


### Example Configurations

Several detailed examples are available [here](/docs/types/severity.md).
