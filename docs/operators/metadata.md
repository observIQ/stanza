## `metadata` operator

The `metadata` operator adds attributes to incoming entries.

### Configuration Fields

| Field      | Default          | Description                                                                                     |
| ---        | ---              | ---                                                                                             |
| `id`       | `metadata`       | A unique identifier for the operator                                                            |
| `output`   | Next in pipeline | The connected operator(s) that will receive all outbound entries                                |
| `attributes`   | {}               | A map of `key: value` attributes to add to the entry's attributes                                       |
| `resource` | {}               | A map of `key: value` attributes to add to the entry's resource                                     |
| `on_error` | `send`           | The behavior of the operator if it encounters an error. See [on_error](/docs/types/on_error.md) |

Inside the attribute values, an [expression](/docs/types/expression.md) surrounded by `EXPR()`
will be replaced with the evaluated form of the expression. The entry's body can be accessed
with the `$` variable in the expression so attributes can be added dynamically from fields.

### Example Configurations


#### Add static attributes and resource

Configuration:
```yaml
- type: metadata
  attributes:
    environment: "production"
  resource:
    cluster: "blue"
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "attributes": {},
  "body": {
    "message": "test"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "attributes": {
    "environment": "production"
  },
  "resource": {
    "cluster": "blue"
  },
  "body": {
    "message": "test"
  }
}
```

</td>
</tr>
</table>

#### Add dynamic attributes

Configuration:
```yaml
- type: metadata
  output: metadata_receiver
  attributes:
    environment: 'EXPR( $.environment == "production" ? "prod" : "dev" )'
```

<table>
<tr><td> Input entry </td> <td> Output entry </td></tr>
<tr>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "attributes": {},
  "body": {
    "production_location": "us_east",
    "environment": "nonproduction"
  }
}
```

</td>
<td>

```json
{
  "timestamp": "2020-06-15T11:15:50.475364-04:00",
  "attributes": {
    "environment": "dev"
  },
  "body": {
    "production_location": "us_east",
    "environment": "nonproduction"
  }
}
```

</td>
</tr>
</table>
