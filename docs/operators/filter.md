## `filter` operator

The `filter` operator filters incoming entries that match an expression.

### Configuration Fields

| Field      | Default          | Description                                                                                     |
| ---        | ---              | ---                                                                                             |
| `id`       | `filter`         | A unique identifier for the operator                                                            |
| `output`   | Next in pipeline | The connected operator(s) that will receive all outbound entries                                |
| `expr`     | required         | An [expression](/docs/types/expression.md) that filters matching entries                        |

### Examples

#### Filter entries based on a regex pattern

```yaml
- type: filter
  expr: '$record.message matches "^LOG: .* END$"'
  output: my_output
```

#### Filter entries based on a label value

```yaml
- type: filter
  expr: '$labels.env == "production"'
  output: my_output
```

#### Filter entries based on an environment variable

```yaml
- type: filter
  expr: '$record.message == env("MY_ENV_VARIABLE")'
  output: my_output
```
