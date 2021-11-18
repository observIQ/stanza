## `severity_parser` operator

The `severity_parser` operator sets the severity on an entry by parsing a value from the body.

### Configuration Fields

| Field         | Default   | Description                                                                                                                                                                                                                            |
| ---           | ---       | ---                                                                                                                                                                                                                                    |
| `id`          | required  | A unique identifier for the operator                                                                                                                                                                                                   |
| `output`      | required  | The `id` for the operator to send parsed entries to                                                                                                                                                                                    |
| `parse_from`  | required  | A [field](/docs/types/field.md) that indicates the field to be parsed as JSON                                                                                                                                                          |
| `preserve_to` |           | Preserves the unparsed value at the specified [field](/docs/types/field.md)                                                                                                                                                            |
| `on_error`    | `send`    | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md)                                                                                                                                        |
| `preset`      | `default` | A predefined set of values that should be interpreted at specific severity levels                                                                                                                                                      |
| `mapping`     |           | A formatted set of values that should be interpreted as severity levels.                                                                                                                                                               |
| `if`          |           | An [expression](/docs/types/expression.md) that, when set, will be evaluated to determine whether this operator should be used for the given entry. This allows you to do easy conditional parsing without branching logic with routers. |


### Example Configurations

Several detailed examples are available [here](/docs/types/severity.md).
